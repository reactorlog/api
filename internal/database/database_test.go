package database

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestOpenPoolInvalidURL(t *testing.T) {
	ctx := context.Background()
	_, err := openPool(ctx, "://bad")
	if err == nil || !strings.Contains(err.Error(), "failed to create pool") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyPoolUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := openPool(ctx, "postgres://reactorlog:reactorlog@127.0.0.1:1/reactorlog?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}

	err = verifyPool(ctx, pool)
	if err == nil || !strings.Contains(err.Error(), "failed to ping database") {
		t.Fatalf("error = %v", err)
	}
}

func TestConnectInvalidURL(t *testing.T) {
	_, err := Connect(context.Background(), "://bad")
	if err == nil || !strings.Contains(err.Error(), "failed to create pool") {
		t.Fatalf("error = %v", err)
	}
}

func TestConnectPingFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := Connect(ctx, "postgres://reactorlog:reactorlog@127.0.0.1:1/reactorlog?connect_timeout=1")
	if err == nil || !strings.Contains(err.Error(), "failed to ping database") {
		t.Fatalf("error = %v", err)
	}
}
