package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/reactorlog/api/internal/database/query"
	"github.com/reactorlog/api/internal/database/schema"
)

var migrationsSchema = schema.Table{
	Name: "migrations",
	Columns: []schema.Column{
		schema.UUID("id").PrimaryKey().DefaultUUIDV7(),
		schema.Int("version").NotNull().Unique(),
		schema.Text("name").NotNull(),
		schema.Timestamp("created_at").
			NotNull().
			DefaultCurrentTimestamp(),
	},
}

func migrationsTableSQL() (string, error) {
	return schema.Build(migrationsSchema)
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	query, err := migrationsTableSQL()
	if err != nil {
		return fmt.Errorf("build migrations table: %w", err)
	}
	_, err = db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	return nil
}

func appliedVersionsQuery() (string, []any, error) {
	return query.Select("version").From("migrations").Build()
}

func scanVersions(rows *sql.Rows) (map[int]bool, error) {
	versions := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan applied version: %w", err)
		}
		versions[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return versions, nil
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[int]bool, error) {
	sql, args, err := appliedVersionsQuery()
	if err != nil {
		return nil, fmt.Errorf("build applied versions query: %w", err)
	}

	rows, err := db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query applied versions: %w", err)
	}
	defer rows.Close()

	return scanVersions(rows)
}

func commitMigration(ctx context.Context, tx *sql.Tx, migration Migration) error {
	if err := migration.Up(ctx, tx); err != nil {
		return fmt.Errorf("up migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func runMigration(ctx context.Context, db *sql.DB, migration Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	return commitMigration(ctx, tx, migration)
}

func applyPending(ctx context.Context, db *sql.DB, migrations []Migration, applied map[int]bool) error {
	for _, migration := range migrations {
		if applied[migration.Version] {
			continue
		}
		if err := runMigration(ctx, db, migration); err != nil {
			return fmt.Errorf(
				"run migration %d %q: %w",
				migration.Version,
				migration.Name,
				err,
			)
		}
	}
	return nil
}

func Run(ctx context.Context, db *sql.DB, migrations []Migration) error {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}

	return applyPending(ctx, db, migrations, applied)
}
