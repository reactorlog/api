package database

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestVerificationPoolClosed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	pool, err := openPool(ctx, "postgres://test:test@127.0.0.1:1/test?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if verifyPool(ctx, pool) == nil {
		t.Fatal("expected ping failure")
	}
	_, err = pool.Acquire(ctx)
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("pool still open: %v", err)
	}
}
