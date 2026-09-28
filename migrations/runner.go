package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/reactorlog/api/internal/database/query"
	"github.com/reactorlog/api/internal/database/schema"
)

var migrationsSchema = schema.Table{
	Name: "migrations",
	IfNotExists: true,
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

func recordMigration(
	ctx context.Context,
	tx *sql.Tx,
	migration Migration,
) error {
	sql, args, err := query.
		Insert("migrations").
		Columns("version", "name").
		Values(migration.Version, migration.Name).
		Build()
	if err != nil {
		return fmt.Errorf("build migration record: %w", err)
	}
	_, err = tx.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("record migration: %w", err)
	}
	return nil
}

func validateMigration(migrations []Migration) error {
	seen := make(map[int]bool, len(migrations))
	for i, migration := range migrations {
		if err := validateMigrationEntry(i, migration); err != nil {
			return err
		}
		if seen[migration.Version] {
			return fmt.Errorf("migration %d: version %d already exists", i, migration.Version)
		}
		seen[migration.Version] = true
	}
	return nil
}

func validateMigrationEntry(i int, migration Migration) error {
	if migration.Version <= 0 {
		return fmt.Errorf("migration %d version must be positive", i)
	}
	if migration.Name == "" {
		return fmt.Errorf("migration %d: name is required", i)
	}
	if migration.Up == nil {
		return fmt.Errorf("migration %d: up function is required", i)
	}
	if migration.Down == nil {
		return fmt.Errorf("migration %d: down function is required", i)
	}
	return nil
}

func runMigration(ctx context.Context, db *sql.DB, migration Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := migration.Up(ctx, tx); err != nil {
		return fmt.Errorf("up migration: %w", err)
	}
	if err := recordMigration(ctx, tx, migration); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func applyPending(ctx context.Context, db *sql.DB, migrations []Migration, applied map[int]bool) error {
	for _, migration := range migrations {
		if applied[migration.Version] {
			continue
		}
		startedAt := time.Now()
		slog.Info(
			"applying migration",
			"version", migration.Version,
			"name", migration.Name,
		)
		if err := runMigration(ctx, db, migration); err != nil {
			return fmt.Errorf(
				"run migration %d %q: %w",
				migration.Version,
				migration.Name,
				err,
			)
		}

		slog.Info(
			"migration applied",
			"version", migration.Version,
			"name", migration.Name,
			"duration", time.Since(startedAt),
		)
	}
	return nil
}

func Run(ctx context.Context, db *sql.DB, migrations []Migration) error {
	if err := validateMigration(migrations); err != nil {
		return fmt.Errorf("validate migrations: %w", err)
	}
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}

	return applyPending(ctx, db, migrations, applied)
}
