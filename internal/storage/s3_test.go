package storage_test

import (
	"strings"
	"testing"

	"github.com/reactorlog/api/internal/storage"
)

func TestConfigFromEnvRequiresAWSVars(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_ENDPOINT_URL_S3", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("STORAGE_BUCKET", "")

	_, err := storage.ConfigFromEnv()
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, key := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_ENDPOINT_URL_S3", "AWS_REGION"} {
		if !strings.Contains(msg, key) {
			t.Fatalf("error %q missing %s", msg, key)
		}
	}
}

func TestConfigFromEnvDefaultsBucket(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_ENDPOINT_URL_S3", "https://example.neon.tech")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("STORAGE_BUCKET", "")

	cfg, err := storage.ConfigFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Bucket != "excel" {
		t.Fatalf("bucket = %q", cfg.Bucket)
	}
}
