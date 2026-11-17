package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/reactorlog/api/internal/api"
)

type fakeStore struct {
	pingErr     error
	sites       []api.Site
	sitesErr    error
	site        api.SiteDetail
	siteErr     error
	reactors    []api.Reactor
	reactorsErr error
}

func (f *fakeStore) Ping(context.Context) error { return f.pingErr }

func (f *fakeStore) ListSites(context.Context) ([]api.Site, error) {
	if f.sitesErr != nil {
		return nil, f.sitesErr
	}
	if f.sites == nil {
		return []api.Site{}, nil
	}
	return f.sites, nil
}

func (f *fakeStore) SiteByID(context.Context, string) (api.SiteDetail, error) {
	return f.site, f.siteErr
}

func (f *fakeStore) ListReactors(context.Context) ([]api.Reactor, error) {
	if f.reactorsErr != nil {
		return nil, f.reactorsErr
	}
	if f.reactors == nil {
		return []api.Reactor{}, nil
	}
	return f.reactors, nil
}

func TestHandlerStatusMapping(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	site := api.Site{ID: "11111111-1111-1111-1111-111111111111", Name: "Example", CreatedAt: now, UpdatedAt: now}
	detail := api.SiteDetail{Site: site, Reactors: []api.Reactor{}}

	tests := []struct {
		name   string
		store  *fakeStore
		method string
		path   string
		want   int
	}{
		{name: "health ok", store: &fakeStore{}, method: http.MethodGet, path: "/health", want: http.StatusOK},
		{name: "health error", store: &fakeStore{pingErr: errors.New("down")}, method: http.MethodGet, path: "/health", want: http.StatusServiceUnavailable},
		{name: "sites empty", store: &fakeStore{}, method: http.MethodGet, path: "/sites", want: http.StatusOK},
		{name: "sites error", store: &fakeStore{sitesErr: errors.New("db")}, method: http.MethodGet, path: "/sites", want: http.StatusInternalServerError},
		{name: "site hit", store: &fakeStore{site: detail}, method: http.MethodGet, path: "/sites/" + site.ID, want: http.StatusOK},
		{name: "site miss", store: &fakeStore{siteErr: api.ErrNotFound}, method: http.MethodGet, path: "/sites/" + site.ID, want: http.StatusNotFound},
		{name: "site error", store: &fakeStore{siteErr: errors.New("db")}, method: http.MethodGet, path: "/sites/" + site.ID, want: http.StatusInternalServerError},
		{name: "reactors empty", store: &fakeStore{}, method: http.MethodGet, path: "/reactors", want: http.StatusOK},
		{name: "reactors error", store: &fakeStore{reactorsErr: errors.New("db")}, method: http.MethodGet, path: "/reactors", want: http.StatusInternalServerError},
		{name: "unmatched path", store: &fakeStore{}, method: http.MethodGet, path: "/nope", want: http.StatusNotFound},
		{name: "wrong method", store: &fakeStore{}, method: http.MethodPost, path: "/sites", want: http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			api.NewMux(tt.store).ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestHandlerEmptyListsAreJSONArrays(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sites", nil)
	api.NewMux(&fakeStore{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	var sites []api.Site
	if err := json.Unmarshal(body, &sites); err != nil {
		t.Fatalf("body = %s, error = %v", body, err)
	}
	if sites == nil || len(sites) != 0 {
		t.Fatalf("sites = %#v", sites)
	}
}
