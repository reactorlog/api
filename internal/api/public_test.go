package api_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/reactorlog/api/internal/api"
)

func TestPublicStripsAPIPrefix(t *testing.T) {
	apiMux := api.NewMux(&fakeStore{})
	pub := api.Public{API: apiMux}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	pub.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPublicDoesNotExposeUnprefixedAPI(t *testing.T) {
	apiMux := api.NewMux(&fakeStore{})
	static := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("spa")},
	}
	pub := api.Public{API: apiMux, Static: static}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sites/abc", nil)
	pub.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "spa" {
		t.Fatalf("body = %q, want SPA fallback", body)
	}
}

func TestPublicServesStaticFile(t *testing.T) {
	apiMux := api.NewMux(&fakeStore{})
	static := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("spa")},
		"app.js":     &fstest.MapFile{Data: []byte("js")},
	}
	pub := api.Public{API: apiMux, Static: static}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	pub.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "js") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestPublicLive(t *testing.T) {
	pub := api.Public{API: api.NewMux(&fakeStore{})}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	pub.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestPublicWithoutStatic404(t *testing.T) {
	pub := api.Public{API: api.NewMux(&fakeStore{})}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	pub.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}
