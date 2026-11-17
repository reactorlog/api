package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/reactorlog/api/internal/api"
	"github.com/reactorlog/api/migrations"
)

func testPool(t *testing.T) *pgxpool.Pool {
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
	schema := fmt.Sprintf("api_read_test_%d", time.Now().UnixNano())
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

	poolConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestAPIReadIntegration(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	var siteID string
	err := pool.QueryRow(ctx, `
		insert into sites (name, location, state)
		values ('Integration Site', 'Testville, ZZ', 'ZZ')
		returning id::text
	`).Scan(&siteID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		insert into reactors (site_id, name, unit_number, docket_number, reactor_type)
		values ($1::uuid, 'Integration 1', '1', '05000999', 'PWR')
	`, siteID)
	if err != nil {
		t.Fatal(err)
	}

	mux := api.NewMux(api.NewStore(pool))

	t.Run("list sites and reactors", func(t *testing.T) {
		sitesRec := httptest.NewRecorder()
		mux.ServeHTTP(sitesRec, httptest.NewRequest(http.MethodGet, "/sites", nil))
		if sitesRec.Code != http.StatusOK {
			t.Fatalf("sites status = %d", sitesRec.Code)
		}
		var sites []api.Site
		if err := json.NewDecoder(sitesRec.Body).Decode(&sites); err != nil {
			t.Fatal(err)
		}
		if len(sites) != 1 || sites[0].ID != siteID || sites[0].Timezone != nil {
			t.Fatalf("sites = %#v", sites)
		}

		reactorsRec := httptest.NewRecorder()
		mux.ServeHTTP(reactorsRec, httptest.NewRequest(http.MethodGet, "/reactors", nil))
		if reactorsRec.Code != http.StatusOK {
			t.Fatalf("reactors status = %d", reactorsRec.Code)
		}
		var reactors []api.Reactor
		if err := json.NewDecoder(reactorsRec.Body).Decode(&reactors); err != nil {
			t.Fatal(err)
		}
		if len(reactors) != 1 || reactors[0].SiteID != siteID || reactors[0].Design != nil {
			t.Fatalf("reactors = %#v", reactors)
		}
	})

	t.Run("get site hit and miss", func(t *testing.T) {
		hit := httptest.NewRecorder()
		mux.ServeHTTP(hit, httptest.NewRequest(http.MethodGet, "/sites/"+siteID, nil))
		if hit.Code != http.StatusOK {
			t.Fatalf("hit status = %d", hit.Code)
		}
		var detail api.SiteDetail
		if err := json.NewDecoder(hit.Body).Decode(&detail); err != nil {
			t.Fatal(err)
		}
		if detail.ID != siteID || len(detail.Reactors) != 1 || detail.Reactors[0].DocketNumber != "05000999" {
			t.Fatalf("detail = %#v", detail)
		}

		missID := "00000000-0000-0000-0000-000000000000"
		miss := httptest.NewRecorder()
		mux.ServeHTTP(miss, httptest.NewRequest(http.MethodGet, "/sites/"+missID, nil))
		if miss.Code != http.StatusNotFound {
			t.Fatalf("miss status = %d", miss.Code)
		}
	})
}
