package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/reactorlog/api/migrations"
)

func TestExitCode(t *testing.T) {
	t.Run("missing env file", func(t *testing.T) {
		t.Chdir(t.TempDir())

		var code int
		stderr := capture(t, &os.Stderr, func() {
			code = exitCode(context.Background(), nil)
		})
		if code != 1 {
			t.Fatalf("code = %d", code)
		}
		if !strings.Contains(stderr, "load .env") {
			t.Fatalf("stderr = %q", stderr)
		}
	})

	t.Run("command error", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ".env"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Chdir(dir)

		var code int
		stderr := capture(t, &os.Stderr, func() {
			code = exitCode(context.Background(), []string{"nope"})
		})
		if code != 1 {
			t.Fatalf("code = %d", code)
		}
		if !strings.Contains(stderr, `unknown command "nope"`) {
			t.Fatalf("stderr = %q", stderr)
		}
	})
}

func TestRun(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	err := run(context.Background(), nil)
	if err == nil || err.Error() != "command required: server or migrate" {
		t.Fatalf("error = %v", err)
	}

	err = run(context.Background(), []string{"nope"})
	if err == nil || err.Error() != `unknown command "nope"` {
		t.Fatalf("error = %v", err)
	}

	err = run(context.Background(), []string{"server"})
	if err == nil || err.Error() != "DATABASE_URL is required" {
		t.Fatalf("error = %v", err)
	}

	err = run(context.Background(), []string{"migrate"})
	if err == nil || err.Error() != "DATABASE_URL is required" {
		t.Fatalf("error = %v", err)
	}
}

func TestRunServer(t *testing.T) {
	t.Run("missing database url", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		err := runServer(context.Background())
		if err == nil || err.Error() != "DATABASE_URL is required" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("connect error", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://127.0.0.1:1/reactorlog?sslmode=disable")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		err := runServer(ctx)
		if err == nil {
			t.Fatal("expected connect error")
		}
	})
}

func TestRunMigrations(t *testing.T) {
	t.Run("missing database url", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		err := runMigrations(context.Background())
		if err == nil || err.Error() != "DATABASE_URL is required" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("run error", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://127.0.0.1:1/reactorlog?sslmode=disable")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		err := runMigrations(ctx)
		if err == nil || !strings.Contains(err.Error(), "run migrations:") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestPrintMigrationResults(t *testing.T) {
	results := []migrations.Result{
		{Version: 1, Name: "create_sites", Applied: false},
		{Version: 2, Name: "create_reactors", Applied: true, Duration: 200 * time.Microsecond},
	}

	out := capture(t, &os.Stdout, func() {
		printMigrationResults(results, time.Millisecond)
	})
	for _, want := range []string{"create_sites", "already applied", "create_reactors", "complete"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output = %q, missing %q", out, want)
		}
	}
}

func capture(t *testing.T, target **os.File, fn func()) string {
	t.Helper()

	original := *target
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	*target = writer
	defer func() {
		*target = original
		reader.Close()
		writer.Close()
	}()

	fn()

	*target = original
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}
