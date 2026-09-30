package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	os.Exit(exitCode(context.Background(), os.Args[1:]))
}

func exitCode(ctx context.Context, args []string) int {
	if err := godotenv.Load(); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("load .env: %w", err))
		return 1
	}

	if err := run(ctx, args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("command required: server or migrate")
	}

	switch args[0] {
	case "server":
		return runServer(ctx)

	case "migrate":
		return runMigrations(ctx)

	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
