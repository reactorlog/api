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

func TestHandlerJSONIncludesReactorCount(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	site := api.Site{
		ID:           "11111111-1111-1111-1111-111111111111",
		Name:         "Example",
		ReactorCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	listRec := httptest.NewRecorder()
	api.NewMux(&fakeStore{sites: []api.Site{site}}).ServeHTTP(
		listRec,
		httptest.NewRequest(http.MethodGet, "/sites", nil),
	)
	if listRec.Code != http.StatusOK {
		t.Fatalf("status = %d", listRec.Code)
	}
	var listed []map[string]any
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0]["reactor_count"] != float64(0) {
		t.Fatalf("sites = %s", listRec.Body.Bytes())
	}

	site.ReactorCount = 1
	detailRec := httptest.NewRecorder()
	api.NewMux(&fakeStore{site: api.SiteDetail{
		Site: site,
		Reactors: []api.Reactor{{
			ID:     "22222222-2222-2222-2222-222222222222",
			SiteID: site.ID,
			Name:   "Example 1",
		}},
	}}).ServeHTTP(detailRec, httptest.NewRequest(http.MethodGet, "/sites/"+site.ID, nil))
	var got map[string]any
	if err := json.Unmarshal(detailRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	reactors, _ := got["reactors"].([]any)
	if got["reactor_count"] != float64(1) || len(reactors) != 1 {
		t.Fatalf("detail = %s", detailRec.Body.Bytes())
	}
}

func TestReactorJSONCapacity(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	mw := 1138.5
	reactor := api.Reactor{
		ID:           "22222222-2222-2222-2222-222222222222",
		SiteID:       "11111111-1111-1111-1111-111111111111",
		Name:         "Example 1",
		UnitNumber:   "1",
		DocketNumber: "05000001",
		ReactorType:  "PWR",
		CapacityMW:   &mw,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	rec := httptest.NewRecorder()
	api.NewMux(&fakeStore{reactors: []api.Reactor{reactor}}).ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodGet, "/reactors", nil),
	)
	var listed []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0]["capacity_mw"] != 1138.5 {
		t.Fatalf("reactors = %s", rec.Body.Bytes())
	}

	reactor.CapacityMW = nil
	empty := httptest.NewRecorder()
	api.NewMux(&fakeStore{reactors: []api.Reactor{reactor}}).ServeHTTP(
		empty,
		httptest.NewRequest(http.MethodGet, "/reactors", nil),
	)
	var missing []map[string]any
	if err := json.Unmarshal(empty.Body.Bytes(), &missing); err != nil {
		t.Fatal(err)
	}
	if len(missing) != 1 || missing[0]["capacity_mw"] != nil {
		t.Fatalf("reactors = %s", empty.Body.Bytes())
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
