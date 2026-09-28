package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/reactorlog/api/migrations"
)

func runMigrations(ctx context.Context) error {
	startedAt := time.Now()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	slog.Info("running migrations")

	if err := migrations.Run(ctx, db, migrations.All); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	slog.Info(
		"migrations complete", 
		"duration", 
		time.Since(startedAt),
	)

	return nil
}
