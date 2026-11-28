package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reactorlog/api/internal/api"
)

func TestParseCORSOrigins(t *testing.T) {
	if got := api.ParseCORSOrigins(""); got != nil {
		t.Fatalf("empty = %#v", got)
	}
	got := api.ParseCORSOrigins(" https://www.reactorlog.com ,https://example.com ")
	if len(got) != 2 || got[0] != "https://www.reactorlog.com" || got[1] != "https://example.com" {
		t.Fatalf("got %#v", got)
	}
}

func TestWithCORSAllowsListedOrigin(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})
	h := api.WithCORS(inner, []string{"https://www.reactorlog.com"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sites", nil)
	req.Header.Set("Origin", "https://www.reactorlog.com")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://www.reactorlog.com" {
		t.Fatalf("ACAO = %q", got)
	}
}

func TestWithCORSDeniesUnknownOrigin(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := api.WithCORS(inner, []string{"https://www.reactorlog.com"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sites", nil)
	req.Header.Set("Origin", "https://evil.example")
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("ACAO = %q, want empty", got)
	}
}

func TestWithCORSOptions(t *testing.T) {
	inner := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("inner should not run for OPTIONS")
	})
	h := api.WithCORS(inner, []string{"https://www.reactorlog.com"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/sites", nil)
	req.Header.Set("Origin", "https://www.reactorlog.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, OPTIONS" {
		t.Fatalf("ACAM = %q", got)
	}
}
