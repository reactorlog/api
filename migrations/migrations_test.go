package migrations

import (
	"context"
	"database/sql"
	"os"
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
