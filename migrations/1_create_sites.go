package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/reactorlog/api/internal/database/schema"
)

var sites = schema.Table{
	Name: "sites",
	Columns: []schema.Column{
		schema.UUID("id").PrimaryKey().DefaultUUIDV7(),
		schema.Text("eia_plant_code"),
		schema.Text("name").NotNull(),
		schema.Text("timezone").NotNull(),
		schema.Float("latitude").Between(-90, 90),
		schema.Float("longitude").Between(-180, 180),
		schema.Text("address_line_1"),
		schema.Text("address_line_2"),
		schema.Text("city"),
		schema.Text("state"),
		schema.Text("zip"),
		schema.Text("phone"),
		schema.Text("email"),
		schema.Timestampz("created_at").NotNull().DefaultCurrentTimestamp(),
		schema.Timestampz("updated_at").NotNull().DefaultCurrentTimestamp(),
	},
}

func sitesCreateSQL() (string, error) {
	return schema.Build(sites)
}

func sitesDropSQL() (string, error) {
	return schema.Drop(sites)
}

func createSites(ctx context.Context, tx *sql.Tx) error {
	query, err := sitesCreateSQL()
	if err != nil {
		return fmt.Errorf("build sites table: %w", err)
	}
	_, err = tx.ExecContext(ctx, query)
	return err
}

func dropSites(ctx context.Context, tx *sql.Tx) error {
	query, err := sitesDropSQL()
	if err != nil {
		return fmt.Errorf("drop sites table: %w", err)
	}
	_, err = tx.ExecContext(ctx, query)
	return err
}

var createSitesMigration = Migration{
	Version: 1,
	Name:    "create_sites",
	Up:      createSites,
	Down:    dropSites,
}
