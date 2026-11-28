package eia

import (
	"context"
	"database/sql"
	"database/sql/driver"
	_ "embed"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

//go:embed sql/advisory_lock.sql
var advisoryLockSQL string

//go:embed sql/enrich.sql
var enrichSQL string

//go:embed sql/reactors_for_sites.sql
var reactorsForSitesSQL string

//go:embed sql/capacity.sql
var capacitySQL string

// DBTX is the database surface Import needs.
type DBTX interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// ImportResult is what a successful Import did.
type ImportResult struct {
	Matched         int
	Updated         int
	ReactorsUpdated int
	Skipped         int
	ByMethod        map[MatchMethod]int
}

// Import fetches EIA-860 nuclear plants, matches them to sites, and writes enrichment.
func Import(ctx context.Context, db DBTX) (ImportResult, error) {
	plants, generators, err := Fetch(ctx)
	if err != nil {
		return ImportResult{}, err
	}
	return store(ctx, db, plants, generators)
}

func store(ctx context.Context, db DBTX, plants []Plant, generators map[string][]Generator) (ImportResult, error) {
	sites, err := listSites(ctx, db)
	if err != nil {
		return ImportResult{}, err
	}
	result, err := Match(sites, plants)
	if err != nil {
		return ImportResult{}, fmt.Errorf("match EIA plants: %w", err)
	}
	if err := requireFullMatch(result); err != nil {
		return ImportResult{}, err
	}
	logMatchResult(result)
	updated, reactorsUpdated, err := commitEnrichment(ctx, db, result.Matched, generators)
	if err != nil {
		return ImportResult{}, err
	}
	out := ImportResult{
		Matched:         len(result.Matched),
		Updated:         updated,
		ReactorsUpdated: reactorsUpdated,
		Skipped:         len(result.UnmatchedPlants),
		ByMethod:        countMethods(result.Matched),
	}
	logImportComplete(out)
	return out, nil
}

func requireFullMatch(result MatchResult) error {
	if len(result.UnmatchedSites) == 0 {
		return nil
	}
	return fmt.Errorf(
		"unmatched sites (%d): %s",
		len(result.UnmatchedSites),
		strings.Join(result.UnmatchedSites, ", "),
	)
}

func countMethods(matched []MatchedPlant) map[MatchMethod]int {
	byMethod := map[MatchMethod]int{}
	for _, m := range matched {
		byMethod[m.Method]++
	}
	return byMethod
}

func logMatchResult(result MatchResult) {
	for _, m := range result.Matched {
		slog.Info(
			"eia plant matched",
			"site", m.SiteName,
			"plant", m.Plant.Name,
			"eia_plant_code", m.Plant.Code,
			"method", string(m.Method),
		)
	}
	for _, plant := range result.UnmatchedPlants {
		slog.Warn("eia plant unused", "plant", plant.Name, "eia_plant_code", plant.Code)
	}
}

func logImportComplete(out ImportResult) {
	slog.Info(
		"eia plant import complete",
		"matched", out.Matched,
		"updated", out.Updated,
		"reactors_updated", out.ReactorsUpdated,
		"skipped_plants", out.Skipped,
		"exact", out.ByMethod[MatchExact],
		"soft", out.ByMethod[MatchSoft],
		"alias", out.ByMethod[MatchAlias],
	)
}

func commitEnrichment(ctx context.Context, db DBTX, matched []MatchedPlant, generators map[string][]Generator) (int, int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("begin EIA import: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, advisoryLockSQL); err != nil {
		return 0, 0, fmt.Errorf("lock EIA import: %w", err)
	}
	updated, reactorsUpdated, err := writeMatched(ctx, tx, matched, generators)
	if err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit EIA import: %w", err)
	}
	return updated, reactorsUpdated, nil
}

func writeMatched(ctx context.Context, tx *sql.Tx, matched []MatchedPlant, generators map[string][]Generator) (int, int, error) {
	units, err := listReactorUnits(ctx, tx, siteNames(matched))
	if err != nil {
		return 0, 0, err
	}
	rows, err := assignments(matched, units, generators)
	if err != nil {
		return 0, 0, err
	}
	updated, err := writeEnrichment(ctx, tx, matched)
	if err != nil {
		return 0, 0, err
	}
	reactorsUpdated, err := writeCapacity(ctx, tx, rows)
	if err != nil {
		return 0, 0, err
	}
	return updated, reactorsUpdated, nil
}

func siteNames(matched []MatchedPlant) []string {
	names := make([]string, len(matched))
	for i, matchedPlant := range matched {
		names[i] = matchedPlant.SiteName
	}
	return names
}

func listSites(ctx context.Context, db DBTX) ([]string, error) {
	rows, err := db.QueryContext(ctx, `select name from sites order by name`)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	defer rows.Close()
	return scanSiteNames(rows)
}

func scanSiteNames(rows *sql.Rows) ([]string, error) {
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan site: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no sites in database; run ingest-nrc first")
	}
	return names, nil
}

func writeEnrichment(ctx context.Context, tx *sql.Tx, matched []MatchedPlant) (int, error) {
	if len(matched) == 0 {
		return 0, nil
	}
	args := enrichArgs(matched)
	res, err := tx.ExecContext(ctx, enrichSQL, args...)
	if err != nil {
		return 0, fmt.Errorf("enrich sites from EIA: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func enrichArgs(matched []MatchedPlant) []any {
	site := make(textArray, len(matched))
	code := make(textArray, len(matched))
	address := make(textArray, len(matched))
	city := make(textArray, len(matched))
	state := make(textArray, len(matched))
	zip := make(textArray, len(matched))
	lat := make(floatArray, len(matched))
	lon := make(floatArray, len(matched))
	for i, m := range matched {
		site[i] = m.SiteName
		code[i] = m.Plant.Code
		address[i] = m.Plant.AddressLine1
		city[i] = m.Plant.City
		state[i] = m.Plant.State
		zip[i] = m.Plant.Zip
		lat[i] = m.Plant.Latitude
		lon[i] = m.Plant.Longitude
	}
	return []any{site, code, address, city, state, zip, lat, lon}
}

type textArray []string

func (a textArray) Value() (driver.Value, error) {
	return formatTextArray(a), nil
}

func formatTextArray(a []string) string {
	var b strings.Builder
	b.WriteByte('{')
	for i, value := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		writeQuoted(&b, value)
	}
	b.WriteByte('}')
	return b.String()
}

type floatArray []float64

func (a floatArray) Value() (driver.Value, error) {
	return formatFloatArray(a), nil
}

func formatFloatArray(a []float64) string {
	var b strings.Builder
	b.WriteByte('{')
	for i, value := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
	}
	b.WriteByte('}')
	return b.String()
}

func writeQuoted(b *strings.Builder, value string) {
	b.WriteByte('"')
	for _, r := range value {
		writeQuotedRune(b, r)
	}
	b.WriteByte('"')
}

func writeQuotedRune(b *strings.Builder, r rune) {
	if r == '\\' || r == '"' {
		b.WriteByte('\\')
	}
	b.WriteRune(r)
}
