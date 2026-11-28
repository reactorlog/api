package eia

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

type reactorUnit struct {
	SiteName   string
	UnitNumber string
}

type capacityRow struct {
	SiteName   string
	UnitNumber string
	CapacityMW float64
}

func assignments(matched []MatchedPlant, units []reactorUnit, generators map[string][]Generator) ([]capacityRow, error) {
	var rows []capacityRow
	for _, matchedPlant := range matched {
		siteRows, err := rowsForMatch(matchedPlant, unitsForSite(units, matchedPlant.SiteName), generators[matchedPlant.Plant.Code])
		if err != nil {
			return nil, err
		}
		rows = append(rows, siteRows...)
	}
	sortCapacityRows(rows)
	return rows, nil
}

func unitsForSite(units []reactorUnit, site string) []reactorUnit {
	var out []reactorUnit
	for _, unit := range units {
		if unit.SiteName == site {
			out = append(out, unit)
		}
	}
	return out
}

func rowsForMatch(matched MatchedPlant, units []reactorUnit, gens []Generator) ([]capacityRow, error) {
	byID, err := nameplates(matched.Plant.Code, gens)
	if err != nil {
		return nil, err
	}
	unitSet, err := unitSet(matched.SiteName, units)
	if err != nil {
		return nil, err
	}
	if err := requireSameIDs(matched, unitSet, byID); err != nil {
		return nil, err
	}
	rows := make([]capacityRow, 0, len(byID))
	for id, mw := range byID {
		rows = append(rows, capacityRow{SiteName: matched.SiteName, UnitNumber: id, CapacityMW: mw})
	}
	return rows, nil
}

func nameplates(code string, gens []Generator) (map[string]float64, error) {
	out := make(map[string]float64, len(gens))
	for _, generator := range gens {
		if _, ok := out[generator.ID]; ok {
			return nil, fmt.Errorf("plant %s: duplicate generator id %s", code, generator.ID)
		}
		out[generator.ID] = generator.NameplateMW
	}
	return out, nil
}

func unitSet(site string, units []reactorUnit) (map[string]struct{}, error) {
	out := make(map[string]struct{}, len(units))
	for _, unit := range units {
		if _, ok := out[unit.UnitNumber]; ok {
			return nil, fmt.Errorf("site %s: duplicate unit number %s", site, unit.UnitNumber)
		}
		out[unit.UnitNumber] = struct{}{}
	}
	return out, nil
}

func requireSameIDs(matched MatchedPlant, units map[string]struct{}, gens map[string]float64) error {
	if sameKeys(units, gens) {
		return nil
	}
	return fmt.Errorf(
		"site %s plant %s: reactor units [%s] != generator ids [%s]",
		matched.SiteName,
		matched.Plant.Code,
		strings.Join(keysOfSet(units), ", "),
		strings.Join(keysOfMW(gens), ", "),
	)
}

func sameKeys(units map[string]struct{}, gens map[string]float64) bool {
	if len(units) != len(gens) {
		return false
	}
	for id := range units {
		if _, ok := gens[id]; !ok {
			return false
		}
	}
	return true
}

func keysOfSet(units map[string]struct{}) []string {
	ids := make([]string, 0, len(units))
	for id := range units {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func keysOfMW(gens map[string]float64) []string {
	ids := make([]string, 0, len(gens))
	for id := range gens {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortCapacityRows(rows []capacityRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].SiteName != rows[j].SiteName {
			return rows[i].SiteName < rows[j].SiteName
		}
		return rows[i].UnitNumber < rows[j].UnitNumber
	})
}

func listReactorUnits(ctx context.Context, tx *sql.Tx, sites []string) ([]reactorUnit, error) {
	rows, err := tx.QueryContext(ctx, reactorsForSitesSQL, textArray(sites))
	if err != nil {
		return nil, fmt.Errorf("list reactors for capacity: %w", err)
	}
	defer rows.Close()
	return scanReactorUnits(rows)
}

func scanReactorUnits(rows *sql.Rows) ([]reactorUnit, error) {
	var units []reactorUnit
	for rows.Next() {
		var unit reactorUnit
		if err := rows.Scan(&unit.SiteName, &unit.UnitNumber); err != nil {
			return nil, fmt.Errorf("scan reactor unit: %w", err)
		}
		units = append(units, unit)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return units, nil
}

func writeCapacity(ctx context.Context, tx *sql.Tx, rows []capacityRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	res, err := tx.ExecContext(ctx, capacitySQL, capacityArgs(rows)...)
	if err != nil {
		return 0, fmt.Errorf("update reactor capacity: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func capacityArgs(rows []capacityRow) []any {
	sites := make(textArray, len(rows))
	units := make(textArray, len(rows))
	mw := make(floatArray, len(rows))
	for i, row := range rows {
		sites[i] = row.SiteName
		units[i] = row.UnitNumber
		mw[i] = row.CapacityMW
	}
	return []any{sites, units, mw}
}
