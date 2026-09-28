package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/reactorlog/api/internal/database"
)

func runServer(ctx context.Context) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	log.Println("connected to database")

	// actual http server comes next.
	return nil
}
