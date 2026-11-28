package nrc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/operating-units.html")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseNRCList(t *testing.T) {
	reactors, err := Parse(strings.NewReader(fixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	if len(reactors) != 95 {
		t.Fatalf("reactors = %d", len(reactors))
	}
	first := reactors[0]
	if first.DocketNumber != "05000313" || first.SiteName != "Arkansas Nuclear" || first.Location != "6 miles WNW of Russellville, AR" || first.UnitNumber != "1" || first.State != "AR" || first.ReactorType != "PWR" {
		t.Fatalf("first = %+v", first)
	}
	if first.SiteName != reactors[1].SiteName || first.Location != reactors[1].Location {
		t.Fatal("units at the same site have different names or locations")
	}
	foundSingle, foundPart52 := false, false
	for _, r := range reactors {
		if r.Name == "Callaway" {
			foundSingle = r.SiteName == "Callaway" && r.UnitNumber == "1"
		}
		if r.Name == "Vogtle 4" {
			foundPart52 = r.DocketNumber == "05200026" && r.UnitNumber == "4"
		}
	}
	if !foundSingle || !foundPart52 {
		t.Fatal("missing single-unit or Part 52 reactor")
	}
}

func TestParseRejectsInvalidSource(t *testing.T) {
	source := fixture(t)
	for name, input := range map[string]string{
		"error page":       "<html>Access denied</html>",
		"changed header":   strings.Replace(source, "Plant Name", "Facility", 1),
		"missing docket":   strings.Replace(source, "05000313", "unknown", 1),
		"duplicate docket": strings.Replace(source, "05000368", "05000313", 1),
		"unknown type":     strings.Replace(source, ">PWR<", ">unknown<", 1),
		"missing state":    strings.Replace(source, "AR</p>", "</p>", 1),
		"oversized":        strings.Repeat("x", maxPageSize+1),
	} {
		t.Run(name, func(t *testing.T) {
			if records, err := Parse(strings.NewReader(input)); err == nil || records != nil {
				t.Fatalf("records = %v, error = %v", records, err)
			}
		})
	}
}

func TestFetchURL(t *testing.T) {
	t.Run("status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		}))
		defer server.Close()
		if _, err := fetchURL(context.Background(), server.URL); err == nil || !strings.Contains(err.Error(), "HTTP 418") {
			t.Fatal(err)
		}
	})

	t.Run("page", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(fixture(t)))
		}))
		defer server.Close()
		records, err := fetchURL(context.Background(), server.URL)
		if err != nil || len(records) != 95 {
			t.Fatalf("len=%d err=%v", len(records), err)
		}
	})

	t.Run("user agent", func(t *testing.T) {
		var userAgent string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent = r.Header.Get("User-Agent")
			w.WriteHeader(http.StatusTeapot)
		}))
		defer server.Close()
		if _, err := fetchURL(context.Background(), server.URL); err == nil || !strings.Contains(err.Error(), "HTTP 418") {
			t.Fatal(err)
		}
		if userAgent != UserAgent {
			t.Fatalf("User-Agent = %q, want %q", userAgent, UserAgent)
		}
	})

	t.Run("cancelled", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		defer server.Close()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := fetchURL(ctx, server.URL); err == nil {
			t.Fatal("expected cancelled request to fail")
		}
	})
}

func TestImportStopsWhenFetchFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Import(ctx, nil); err == nil {
		t.Fatal("expected fetch failure")
	}
}
