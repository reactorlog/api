package main

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRunRequiresDatabaseURL(t *testing.T) {
	err := run(context.Background(), func(string) string { return "" }, nil)
	if err == nil || err.Error() != "DATABASE_URL is required" {
		t.Fatalf("error = %v", err)
	}
}

func TestRunConnectError(t *testing.T) {
	err := run(
		context.Background(),
		func(string) string { return "postgres://example" },
		func(context.Context, string) (*pgxpool.Pool, error) {
			return nil, errors.New("boom")
		},
	)
	if err == nil || err.Error() != "boom" {
		t.Fatalf("error = %v", err)
	}
}

func TestRunSuccess(t *testing.T) {
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(io.Discard) })

	// Pool creation requires a parseable URL; Close on an unstarted pool is safe.
	pool, err := pgxpool.New(context.Background(), "postgres://reactorlog:reactorlog@127.0.0.1:1/reactorlog")
	if err != nil {
		t.Fatal(err)
	}

	err = run(
		context.Background(),
		func(string) string { return "postgres://example" },
		func(context.Context, string) (*pgxpool.Pool, error) {
			return pool, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunOrExitSuccess(t *testing.T) {
	log.SetOutput(io.Discard)

	pool, err := pgxpool.New(context.Background(), "postgres://reactorlog:reactorlog@127.0.0.1:1/reactorlog")
	if err != nil {
		t.Fatal(err)
	}

	runOrExit(
		context.Background(),
		func(string) string { return "postgres://example" },
		func(context.Context, string) (*pgxpool.Pool, error) {
			return pool, nil
		},
	)
}
