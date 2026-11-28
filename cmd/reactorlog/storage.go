package main

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/reactorlog/api/internal/storage"
)

func runStorageCheckArgs(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: reactorlog storage-check")
	}
	return runStorageCheck(ctx)
}

func runStorageCheck(ctx context.Context) error {
	cfg, err := storage.ConfigFromEnv()
	if err != nil {
		return err
	}
	client := storage.NewClient(cfg)
	key := fmt.Sprintf("checks/%s.txt", time.Now().UTC().Format("20060102T150405Z"))
	payload := []byte("reactorlog storage-check ok\n")
	if err := client.PutObject(ctx, key, bytes.NewReader(payload), "text/plain"); err != nil {
		return err
	}
	got, err := client.GetObject(ctx, key)
	if err != nil {
		return err
	}
	if !bytes.Equal(got, payload) {
		return fmt.Errorf("storage round-trip mismatch: got %q", got)
	}
	fmt.Printf("storage-check ok  bucket=%s key=%s bytes=%d\n", client.Bucket(), key, len(got))
	return nil
}
