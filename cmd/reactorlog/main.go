package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	os.Exit(exitCode(context.Background(), os.Args[1:]))
}

func exitCode(ctx context.Context, args []string) int {
	if err := loadDotEnv(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := run(ctx, args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func loadDotEnv() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load .env: %w", err)
	}
	return nil
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("command required: server, migrate, or ingest-nrc")
	}
	return runCommand(ctx, args)
}

func runCommand(ctx context.Context, args []string) error {
	switch args[0] {
	case "server":
		return runServer(ctx)
	case "migrate":
		return runMigrations(ctx)
	case "ingest-nrc":
		return runIngestArgs(ctx, args)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runIngestArgs(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: reactorlog ingest-nrc")
	}
	return runNRCIngestion(ctx)
}
