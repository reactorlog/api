package migrations

import (
	"context"
	"database/sql"
	"os"
	"testing"
)

func TestRunFreshDatabase(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	// make the database actually fresh for this test.
	_, err = db.ExecContext(ctx, `
		DROP TABLE IF EXISTS migrations CASCADE;
		DROP TABLE IF EXISTS scrams CASCADE;
		DROP TABLE IF EXISTS units CASCADE;
		DROP TABLE IF EXISTS sites CASCADE;
	`)
	if err != nil {
		t.Fatal(err)
	}

	err = Run(ctx, db, All)
	if err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// we'll add assertions here next.
}