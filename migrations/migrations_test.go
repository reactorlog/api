package migrations

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestRunFreshDatabase(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()

	resetTestDatabase(t, ctx, db)

	results, err := Run(ctx, db, All)
	if err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	if len(results) != len(All) {
		t.Fatalf("results = %d, want %d", len(results), len(All))
	}
	for _, result := range results {
		if !result.Applied {
			t.Fatalf("result %+v not applied on a fresh database", result)
		}
	}

	assertTableExists(t, ctx, db, "migrations")
	assertTableExists(t, ctx, db, "sites")
	assertMigrationCount(t, ctx, db, len(All))
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func resetTestDatabase(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
) {
	t.Helper()

	_, err := db.ExecContext(ctx, `
		DROP TABLE IF EXISTS reactors CASCADE;
		DROP TABLE IF EXISTS migrations CASCADE;
		DROP TABLE IF EXISTS sites CASCADE;
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func assertTableExists(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	table string,
) {
	t.Helper()

	var exists bool

	err := db.QueryRowContext(
		ctx,
		`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name = $1
		)
		`,
		table,
	).Scan(&exists)
	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Errorf("expected table %q to exist", table)
	}
}

func assertMigrationCount(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	want int,
) {
	t.Helper()

	var count int

	err := db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM migrations",
	).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}

	if count != want {
		t.Errorf("migration count = %d, want %d", count, want)
	}
}

func TestMigratedSchema(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()
	resetTestDatabase(t, ctx, db)
	if _, err := Run(ctx, db, All); err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(ctx, `insert into sites (name, location, latitude) values ('Bound', 'place', 91)`); err == nil {
		t.Fatal("latitude 91 was accepted")
	}
	if _, err := db.ExecContext(ctx, `insert into sites (name, location, latitude, longitude) values ('Edge', 'edge', 90, 180)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `insert into sites (name, location) values ('No zone', 'nowhere')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `insert into sites (name, location) values ('No zone', 'nowhere')`); err == nil {
		t.Fatal("duplicate name and location was accepted")
	}
	if _, err := db.ExecContext(ctx, `
		insert into reactors (site_id, name, unit_number, docket_number, reactor_type)
		select id, 'No zone 1', '1', '05000001', 'PWR'
		from sites
		where name = 'No zone'
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		insert into reactors (site_id, name, unit_number, docket_number, reactor_type)
		select id, 'No zone 2', '2', '05000001', 'PWR'
		from sites
		where name = 'No zone'
	`); err == nil {
		t.Fatal("duplicate docket was accepted")
	}
	if _, err := db.ExecContext(ctx, `
		insert into reactors (site_id, name, unit_number, docket_number, reactor_type)
		values ('00000000-0000-0000-0000-000000000000', 'Missing', '1', '05000002', 'PWR')
	`); err == nil {
		t.Fatal("reactor without a site was accepted")
	}

	var indexDef string
	err := db.QueryRowContext(ctx, `
		select pg_get_indexdef(c.oid)
		from pg_class c
		join pg_namespace n on n.oid = c.relnamespace
		where n.nspname = current_schema()
		  and c.relname = 'reactors_docket_number_key'
	`).Scan(&indexDef)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(indexDef, "site_id") {
		t.Fatalf("docket index = %s", indexDef)
	}
}
