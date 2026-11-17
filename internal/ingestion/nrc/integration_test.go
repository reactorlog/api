package nrc

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
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
	schema := fmt.Sprintf("nrc_import_test_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "create schema "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.ExecContext(ctx, "drop schema "+schema+" cascade"); err != nil {
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

func TestReferenceImportDatabase(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	records, err := Parse(strings.NewReader(fixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if count, err := store(ctx, db, records); err != nil || count != 95 {
			t.Fatalf("count=%d, error=%v", count, err)
		}
	}
	var count, sites, expectedSites int
	keys := map[[2]string]bool{}
	for _, r := range records {
		keys[[2]string{r.SiteName, r.Location}] = true
	}
	expectedSites = len(keys)
	if err := db.QueryRow("select count(*) from reactors").Scan(&count); err != nil || count != 95 {
		t.Fatalf("reactors=%d, error=%v", count, err)
	}
	if err := db.QueryRow("select count(*) from sites").Scan(&sites); err != nil || sites != expectedSites {
		t.Fatalf("sites=%d, want=%d, error=%v", sites, expectedSites, err)
	}
	var id string
	if err := db.QueryRow("select id from reactors where docket_number = '05000313'").Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("update reactors set design = 'existing design' where id = $1", id); err != nil {
		t.Fatal(err)
	}
	// A changed source name updates the existing reactor without losing its ID or enrichment.
	records[0].Name = "Updated Arkansas Nuclear 1"
	if _, err := store(ctx, db, records[:1]); err != nil {
		t.Fatal(err)
	}
	var name, design string
	if err := db.QueryRow("select name, design from reactors where id = $1", id).Scan(&name, &design); err != nil {
		t.Fatal(err)
	}
	if name != records[0].Name || design != "existing design" {
		t.Fatalf("name=%s, design=%s", name, design)
	}
	// Renaming a stored site onto another site's name and location must roll back.
	records[0].Name = "Should roll back"
	records[0].SiteName = records[2].SiteName
	records[0].Location = records[2].Location
	if _, err := store(ctx, db, records[:1]); err == nil {
		t.Fatal("expected site identity conflict")
	}
	if err := db.QueryRow("select name from reactors where id = $1", id).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name == "Should roll back" {
		t.Fatal("failed import was partially committed")
	}
}

func TestImportLinksSetwise(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	first := []Reactor{
		{Name: "Plant 1", UnitNumber: "1", DocketNumber: "05000001", ReactorType: "PWR", SiteName: "Plant", Location: "Here", State: "AR"},
		{Name: "Plant 2", UnitNumber: "2", DocketNumber: "05000002", ReactorType: "PWR", SiteName: "Plant", Location: "Here", State: "AR"},
	}
	if count, err := store(ctx, db, first); err != nil || count != 2 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if _, err := db.ExecContext(ctx, `update reactors set design = 'kept', updated_at = '2000-01-01'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `update sites set timezone = 'America/Chicago', updated_at = '2000-01-01'`); err != nil {
		t.Fatal(err)
	}
	if _, err := store(ctx, db, first); err != nil {
		t.Fatal(err)
	}
	var updatedAt, design, timezone string
	if err := db.QueryRowContext(ctx, `select updated_at::text, design from reactors where docket_number = '05000001'`).Scan(&updatedAt, &design); err != nil {
		t.Fatal(err)
	}
	if design != "kept" || !strings.HasPrefix(updatedAt, "2000-01-01") {
		t.Fatalf("design=%s updated_at=%s", design, updatedAt)
	}
	if err := db.QueryRowContext(ctx, `select timezone, updated_at::text from sites`).Scan(&timezone, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if timezone != "America/Chicago" || !strings.HasPrefix(updatedAt, "2000-01-01") {
		t.Fatalf("timezone=%s updated_at=%s", timezone, updatedAt)
	}

	// The new docket is listed before the rename. It still joins the renamed site.
	renamed := []Reactor{
		{Name: "Plant 3", UnitNumber: "3", DocketNumber: "05000003", ReactorType: "PWR", SiteName: "Plant", Location: "There", State: "AR"},
		{Name: "Plant 1", UnitNumber: "1", DocketNumber: "05000001", ReactorType: "PWR", SiteName: "Plant", Location: "There", State: "AR"},
	}
	if _, err := store(ctx, db, renamed); err != nil {
		t.Fatal(err)
	}
	var sites int
	var siteID string
	if err := db.QueryRowContext(ctx, `select count(*) from sites`).Scan(&sites); err != nil || sites != 1 {
		t.Fatalf("sites=%d err=%v", sites, err)
	}
	if err := db.QueryRowContext(ctx, `
		select count(distinct site_id)::text from reactors
	`).Scan(&siteID); err != nil || siteID != "1" {
		t.Fatalf("distinct sites=%s err=%v", siteID, err)
	}
	var newDesign sql.NullString
	if err := db.QueryRowContext(ctx, `select design from reactors where docket_number = '05000003'`).Scan(&newDesign); err != nil {
		t.Fatal(err)
	}
	if newDesign.Valid {
		t.Fatalf("new design=%q", newDesign.String)
	}
}
