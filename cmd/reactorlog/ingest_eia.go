package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/reactorlog/api/internal/ingestion/eia"
)

func runEIAPlantIngestion(ctx context.Context) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	result, err := eia.Import(ctx, db)
	if err != nil {
		return fmt.Errorf("ingest EIA plant data: %w", err)
	}
	fmt.Printf(
		"Enriched %d sites from EIA plants (%d rows updated, %d unused plants).\n",
		result.Matched,
		result.Updated,
		result.Skipped,
	)
	return nil
}
