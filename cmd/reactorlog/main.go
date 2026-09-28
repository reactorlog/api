package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("load .env: %w", err))
		os.Exit(1)
	}

	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
