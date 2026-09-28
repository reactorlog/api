package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/reactorlog/api/internal/database"
)

type (
	getenvFunc  func(key string) string
	connectFunc func(ctx context.Context, databaseURL string) (*pgxpool.Pool, error)
)

func main() {
	runOrExit(context.Background(), os.Getenv, database.Connect)
}

func runOrExit(ctx context.Context, getenv getenvFunc, connect connectFunc) {
	if err := run(ctx, getenv, connect); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, getenv getenvFunc, connect connectFunc) error {
	databaseURL := getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	pool, err := connect(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	log.Println("connected to database")
	return nil
}
