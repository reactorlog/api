package eia

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/reactorlog/api/migrations"
)

func testDatabase(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	config, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	admin := sql.OpenDB(stdlib.GetConnector(*config))
	schema := fmt.Sprintf("eia_capacity_test_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "create schema "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.ExecContext(context.Background(), "drop schema "+schema+" cascade"); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	config.RuntimeParams["search_path"] = schema
	db := sql.OpenDB(stdlib.GetConnector(*config))
	t.Cleanup(func() { db.Close() })
	if _, err := migrations.Run(ctx, db, migrations.All); err != nil {
		t.Fatal(err)
	}
	return db
}

func insertDiablo(t *testing.T, db *sql.DB, unit string) {
	t.Helper()
	ctx := context.Background()
	var siteID string
	if err := db.QueryRowContext(ctx, `insert into sites (name) values ('Diablo Canyon') returning id::text`).Scan(&siteID); err != nil {
		t.Fatal(err)
	}
	_, err := db.ExecContext(ctx, `
		insert into reactors (site_id, name, unit_number, docket_number, reactor_type)
		values ($1::uuid, 'Diablo Canyon 1', $2, '05000323', 'PWR')
	`, siteID, unit)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCapacityPersistsAndSkipsWhenUnchanged(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	insertDiablo(t, db, "1")
	plant := samplePlant("6099", "Diablo Canyon", "Diablo Canyon")
	generators := diabloGenerators(1138.5)

	first, err := store(ctx, db, []Plant{plant}, generators)
	if err != nil || first.Updated != 1 || first.ReactorsUpdated != 1 {
		t.Fatalf("%+v %v", first, err)
	}
	var mw float64
	var updated string
	if err := db.QueryRowContext(ctx, `select capacity_mw, updated_at::text from reactors`).Scan(&mw, &updated); err != nil {
		t.Fatal(err)
	}
	if mw != 1138.5 {
		t.Fatalf("capacity=%v", mw)
	}

	second, err := store(ctx, db, []Plant{plant}, generators)
	if err != nil || second.Updated != 0 || second.ReactorsUpdated != 0 {
		t.Fatalf("%+v %v", second, err)
	}
	var mw2 float64
	var updated2 string
	if err := db.QueryRowContext(ctx, `select capacity_mw, updated_at::text from reactors`).Scan(&mw2, &updated2); err != nil {
		t.Fatal(err)
	}
	if mw2 != 1138.5 || updated2 != updated {
		t.Fatalf("capacity=%v updated_at=%s want %s", mw2, updated2, updated)
	}
}

func TestCapacityMismatchRollsBack(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	insertDiablo(t, db, "2")
	plant := samplePlant("6099", "Diablo Canyon", "Diablo Canyon")
	_, err := store(ctx, db, []Plant{plant}, diabloGenerators(1138.5))
	if err == nil {
		t.Fatal("expected generator mismatch")
	}
	var capacity sql.NullFloat64
	var code sql.NullString
	if err := db.QueryRowContext(ctx, `select capacity_mw, (select eia_plant_code from sites) from reactors`).Scan(&capacity, &code); err != nil {
		t.Fatal(err)
	}
	if capacity.Valid || code.Valid {
		t.Fatalf("capacity=%v code=%v", capacity, code)
	}
}
