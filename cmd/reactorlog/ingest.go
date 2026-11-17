package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/reactorlog/api/internal/ingestion/nrc"
)

func runNRCIngestion(ctx context.Context) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()
	count, err := nrc.Import(ctx, db)
	if err != nil {
		return fmt.Errorf("ingest NRC reference data: %w", err)
	}
	fmt.Printf("Imported %d NRC reactors.\n", count)
	return nil
}
