package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/reactorlog/api/migrations"
)

const (
	green = "\033[32m"
	dim   = "\033[2m"
	reset = "\033[0m"
)

func printMigrationResults(results []migrations.Result, duration time.Duration) {
	fmt.Println("migrations")
	fmt.Println()

	for _, result := range results {
		if !result.Applied {
			fmt.Printf(
				"  %s- %03d  %-24s already applied%s\n",
				dim,
				result.Version,
				result.Name,
				reset,
			)
			continue
		}

		fmt.Printf(
			"  %s✓%s %03d  %-24s %s\n",
			green,
			reset,
			result.Version,
			result.Name,
			result.Duration.Round(100*time.Microsecond),
		)
	}

	fmt.Println()
	fmt.Printf(
		"%s✓ complete%s  %s\n",
		green,
		reset,
		duration.Round(100*time.Microsecond),
	)
}

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

	results, err := migrations.Run(ctx, db, migrations.All)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	printMigrationResults(results, time.Since(startedAt))
	return nil
}
