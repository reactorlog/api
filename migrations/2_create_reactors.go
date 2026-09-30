package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/reactorlog/api/internal/database/schema"
)

var reactors = schema.Table{
	Name: "reactors",
	Columns: []schema.Column{
		schema.UUID("id").PrimaryKey().DefaultUUIDV7(),
		schema.UUID("site_id").
			NotNull().
			References("sites", "id"),
		schema.Text("name").NotNull(),
		schema.Text("unit_number").NotNull(),
		schema.Text("docket_number").NotNull().Unique(),
		schema.Text("reactor_type").NotNull(),
		schema.Text("design").NotNull(),
		schema.Timestampz("created_at").NotNull().DefaultCurrentTimestamp(),
		schema.Timestampz("updated_at").NotNull().DefaultCurrentTimestamp(),
	},
}

func reactorsCreateSQL() (string, error) {
	return schema.Build(reactors)
}

func reactorsDropSQL() (string, error) {
	return schema.Drop(reactors)
}

func createReactors(ctx context.Context, tx *sql.Tx) error {
	query, err := reactorsCreateSQL()
	if err != nil {
		return fmt.Errorf("build reactors table: %w", err)
	}

	_, err = tx.ExecContext(ctx, query)
	return err
}

func dropReactors(ctx context.Context, tx *sql.Tx) error {
	query, err := reactorsDropSQL()
	if err != nil {
		return fmt.Errorf("drop reactors table: %w", err)
	}

	_, err = tx.ExecContext(ctx, query)
	return err
}

var createReactorsMigration = Migration{
	Version: 2,
	Name:    "create_reactors",
	Up:      createReactors,
	Down:    dropReactors,
}
