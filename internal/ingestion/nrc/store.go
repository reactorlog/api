package nrc

import (
	"context"
	"database/sql"
	"database/sql/driver"
	_ "embed"
	"fmt"
	"strings"
)

//go:embed sql/advisory_lock.sql
var advisoryLockSQL string

//go:embed sql/import.sql
var importSQL string

// Import fetches and validates the source before opening a transaction. It never
// deletes records or overwrites metadata that is absent from the source list.
func Import(ctx context.Context, db *sql.DB) (int, error) {
	reactors, err := Fetch(ctx)
	if err != nil {
		return 0, err
	}
	return store(ctx, db, reactors)
}

func store(ctx context.Context, db *sql.DB, reactors []Reactor) (int, error) {
	if len(reactors) == 0 {
		return 0, nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin NRC import: %w", err)
	}
	defer tx.Rollback()
	if err = writeImport(ctx, tx, reactors); err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit NRC import: %w", err)
	}
	return len(reactors), nil
}

func writeImport(ctx context.Context, tx *sql.Tx, reactors []Reactor) error {
	// One lock for the whole write. The import itself is a single statement.
	if _, err := tx.ExecContext(ctx, advisoryLockSQL); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, importSQL, importArgs(reactors)...); err != nil {
		return fmt.Errorf("save NRC reactors: %w", err)
	}
	return nil
}

func importArgs(reactors []Reactor) []any {
	name := make(textArray, len(reactors))
	unit := make(textArray, len(reactors))
	docket := make(textArray, len(reactors))
	kind := make(textArray, len(reactors))
	site := make(textArray, len(reactors))
	location := make(textArray, len(reactors))
	state := make(textArray, len(reactors))
	for i, reactor := range reactors {
		name[i] = reactor.Name
		unit[i] = reactor.UnitNumber
		docket[i] = reactor.DocketNumber
		kind[i] = reactor.ReactorType
		site[i] = reactor.SiteName
		location[i] = reactor.Location
		state[i] = reactor.State
	}
	return []any{name, unit, docket, kind, site, location, state}
}

// textArray is a Postgres text[] literal. database/sql accepts it because Value
// returns text; the statement casts that text to text[].
type textArray []string

func (a textArray) Value() (driver.Value, error) {
	var b strings.Builder
	b.WriteByte('{')
	for i, value := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		writeQuoted(&b, value)
	}
	b.WriteByte('}')
	return b.String(), nil
}

func writeQuoted(b *strings.Builder, value string) {
	b.WriteByte('"')
	for _, r := range value {
		if r == '\\' || r == '"' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('"')
}
