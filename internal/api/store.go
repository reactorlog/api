package api

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound means the requested row does not exist.
var ErrNotFound = errors.New("not found")

//go:embed sql/list_sites.sql
var listSitesSQL string

//go:embed sql/get_site.sql
var getSiteSQL string

//go:embed sql/list_reactors.sql
var listReactorsSQL string

//go:embed sql/list_reactors_for_site.sql
var listReactorsForSiteSQL string

// Store is the read API's data dependency. Handlers never build SQL.
type Store interface {
	Ping(ctx context.Context) error
	ListSites(ctx context.Context) ([]Site, error)
	SiteByID(ctx context.Context, id string) (SiteDetail, error)
	ListReactors(ctx context.Context) ([]Reactor, error)
}

type dbConn interface {
	Ping(ctx context.Context) error
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// PostgresStore runs the embedded SELECTs against a pool.
type PostgresStore struct {
	db dbConn
}

// NewStore returns a Postgres-backed Store.
func NewStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{db: pool}
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

func (s *PostgresStore) ListSites(ctx context.Context) ([]Site, error) {
	rows, err := s.db.Query(ctx, listSitesSQL)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	defer rows.Close()
	return collectSites(rows)
}

func (s *PostgresStore) SiteByID(ctx context.Context, id string) (SiteDetail, error) {
	if !validUUID(id) {
		return SiteDetail{}, ErrNotFound
	}
	return s.siteDetail(ctx, id)
}

func (s *PostgresStore) siteDetail(ctx context.Context, id string) (SiteDetail, error) {
	site, err := loadSite(s.db.QueryRow(ctx, getSiteSQL, id))
	if err != nil {
		return SiteDetail{}, err
	}
	reactors, err := s.listReactorsForSite(ctx, id)
	if err != nil {
		return SiteDetail{}, err
	}
	return SiteDetail{Site: site, Reactors: reactors}, nil
}

func (s *PostgresStore) ListReactors(ctx context.Context) ([]Reactor, error) {
	rows, err := s.db.Query(ctx, listReactorsSQL)
	if err != nil {
		return nil, fmt.Errorf("list reactors: %w", err)
	}
	defer rows.Close()
	return collectReactors(rows, "list reactors")
}

func (s *PostgresStore) listReactorsForSite(ctx context.Context, siteID string) ([]Reactor, error) {
	rows, err := s.db.Query(ctx, listReactorsForSiteSQL, siteID)
	if err != nil {
		return nil, fmt.Errorf("list reactors for site: %w", err)
	}
	defer rows.Close()
	return collectReactors(rows, "list reactors for site")
}

type rowIterator interface {
	Next() bool
	Err() error
	siteScanner
}

func collectSites(rows rowIterator) ([]Site, error) {
	sites := make([]Site, 0)
	for rows.Next() {
		site, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}
	return finishSites(rows, sites)
}

func finishSites(rows rowIterator, sites []Site) ([]Site, error) {
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	return sites, nil
}

func collectReactors(rows rowIterator, label string) ([]Reactor, error) {
	reactors := make([]Reactor, 0)
	for rows.Next() {
		reactor, err := scanReactor(rows)
		if err != nil {
			return nil, err
		}
		reactors = append(reactors, reactor)
	}
	return finishReactors(rows, reactors, label)
}

func finishReactors(rows rowIterator, reactors []Reactor, label string) ([]Reactor, error) {
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	return reactors, nil
}

func loadSite(row siteScanner) (Site, error) {
	site, err := scanSite(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Site{}, ErrNotFound
	}
	if err != nil {
		return Site{}, fmt.Errorf("get site: %w", err)
	}
	return site, nil
}

type siteScanner interface {
	Scan(dest ...any) error
}

func scanSite(row siteScanner) (Site, error) {
	var site Site
	err := row.Scan(
		&site.ID,
		&site.EIAPlantCode,
		&site.Name,
		&site.Timezone,
		&site.Latitude,
		&site.Longitude,
		&site.AddressLine1,
		&site.AddressLine2,
		&site.City,
		&site.State,
		&site.Location,
		&site.Zip,
		&site.Phone,
		&site.Email,
		&site.CreatedAt,
		&site.UpdatedAt,
	)
	if err != nil {
		return Site{}, err
	}
	return site, nil
}

func scanReactor(row siteScanner) (Reactor, error) {
	var reactor Reactor
	err := row.Scan(
		&reactor.ID,
		&reactor.SiteID,
		&reactor.Name,
		&reactor.UnitNumber,
		&reactor.DocketNumber,
		&reactor.ReactorType,
		&reactor.Design,
		&reactor.CreatedAt,
		&reactor.UpdatedAt,
	)
	if err != nil {
		return Reactor{}, fmt.Errorf("scan reactor: %w", err)
	}
	return reactor, nil
}

func validUUID(id string) bool {
	return len(id) == 36 && uuidBodyOK(id)
}

func uuidBodyOK(id string) bool {
	for i, c := range id {
		if !uuidCharOK(i, c) {
			return false
		}
	}
	return true
}

func uuidCharOK(i int, c rune) bool {
	if uuidHyphenAt(i) {
		return c == '-'
	}
	return isHex(c)
}

func uuidHyphenAt(i int) bool {
	return i == 8 || i == 13 || i == 18 || i == 23
}

func isHex(c rune) bool {
	return hexDigit(c) || hexLower(c) || hexUpper(c)
}

func hexDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

func hexLower(c rune) bool {
	return c >= 'a' && c <= 'f'
}

func hexUpper(c rune) bool {
	return c >= 'A' && c <= 'F'
}
