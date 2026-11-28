package eia

import (
	"os"
	"testing"
)

func TestPlantsFromZipLive(t *testing.T) {
	path := os.Getenv("EIA860_ZIP")
	if path == "" {
		t.Skip("set EIA860_ZIP to run against a real archive")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	plants, _, err := PlantsFromZip(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(plants) < 50 {
		t.Fatalf("expected dozens of nuclear plants, got %d", len(plants))
	}
	found := false
	for _, p := range plants {
		if p.Code == "6099" && p.Name == "Diablo Canyon" {
			found = true
			if p.MatchKey != "diablo canyon" || p.State != "CA" {
				t.Fatalf("%+v", p)
			}
		}
	}
	if !found {
		t.Fatal("Diablo Canyon missing")
	}
}
