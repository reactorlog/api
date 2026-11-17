package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	if err := execSQL(ctx, db, migrationsTableSQL); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	return nil
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
	rows, err := db.QueryContext(ctx, appliedVersionsSQL)
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
	_, err := tx.ExecContext(ctx, recordMigrationSQL, migration.Version, migration.Name)
	if err != nil {
		return fmt.Errorf("record migration: %w", err)
	}
	return nil
}

func validateMigrations(migrations []Migration) error {
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
	if migration.Up == "" {
		return fmt.Errorf("migration %d: up sql is required", i)
	}
	if migration.Down == "" {
		return fmt.Errorf("migration %d: down sql is required", i)
	}
	return nil
}

func runMigration(ctx context.Context, db *sql.DB, migration Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := execSQL(ctx, tx, migration.Up); err != nil {
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

func applyPending(ctx context.Context, db *sql.DB, migrations []Migration, applied map[int]bool) ([]Result, error) {
	results := make([]Result, 0, len(migrations))
	for _, migration := range migrations {
		if applied[migration.Version] {
			results = append(results, Result{
				Version: migration.Version,
				Name:    migration.Name,
				Applied: false,
			})
			continue
		}
		startedAt := time.Now()
		slog.Info(
			"applying migration",
			"version", migration.Version,
			"name", migration.Name,
		)
		if err := runMigration(ctx, db, migration); err != nil {
			slog.Error(
				"apply pending migration",
				"version", migration.Version,
				"name", migration.Name,
				"duration", time.Since(startedAt),
				"error", err,
			)
			return nil, fmt.Errorf(
				"apply pending migration %d %q: %w",
				migration.Version,
				migration.Name,
				err,
			)
		}

		duration := time.Since(startedAt)

		slog.Info(
			"migration applied",
			"version", migration.Version,
			"name", migration.Name,
			"duration", duration,
		)

		results = append(results, Result{
			Version:  migration.Version,
			Name:     migration.Name,
			Applied:  true,
			Duration: duration,
		})
	}
	return results, nil
}

func Run(ctx context.Context, db *sql.DB, migrations []Migration) ([]Result, error) {
	if err := validateMigrations(migrations); err != nil {
		return nil, fmt.Errorf("validate migrations: %w", err)
	}
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return nil, fmt.Errorf("ensure migrations table: %w", err)
	}

	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return nil, err
	}

	return applyPending(ctx, db, migrations, applied)
}
