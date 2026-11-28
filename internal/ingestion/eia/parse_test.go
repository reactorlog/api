package eia

import "testing"

func TestPlantsFromTables(t *testing.T) {
	plantRows := [][]string{
		{"title"},
		{"Utility ID", "Utility Name", "Plant Code", "Plant Name", "Street Address", "City", "State", "Zip", "County", "Latitude", "Longitude"},
		{"1", "A", "6099", "Diablo Canyon", "Avila Rd", "Avila Beach", "CA", "93424", "SLO", "35.211536", "-120.855542"},
		{"1", "A", "100", "Coal Plant", "1 Main", "Town", "TX", "75001", "Dallas", "32.0", "-96.0"},
		{"1", "A", "649", "Vogtle", "7821 River Road", "Waynesboro", "GA", "30830", "Burke", "33.1427", "-81.7625"},
	}
	genRows := [][]string{
		{"title"},
		{"x", "y", "Plant Code", "Plant Name", "Energy Source 1"},
		{"", "", "6099", "Diablo Canyon", "NUC"},
		{"", "", "100", "Coal Plant", "BIT"},
		{"", "", "649", "Vogtle", "NUC"},
		{"", "", "649", "Vogtle", "NUC"},
	}
	plants, err := plantsFromTables(plantRows, genRows)
	if err != nil {
		t.Fatal(err)
	}
	if len(plants) != 2 {
		t.Fatalf("got %d plants: %+v", len(plants), plants)
	}
	byCode := map[string]Plant{}
	for _, p := range plants {
		byCode[p.Code] = p
	}
	diablo := byCode["6099"]
	if diablo.Name != "Diablo Canyon" || diablo.City != "Avila Beach" || diablo.State != "CA" || diablo.Zip != "93424" {
		t.Fatalf("%+v", diablo)
	}
	if diablo.Latitude != 35.211536 || diablo.Longitude != -120.855542 {
		t.Fatalf("coords %+v", diablo)
	}
	if diablo.MatchKey != "diablo canyon" {
		t.Fatalf("match key %q", diablo.MatchKey)
	}
	if _, ok := byCode["100"]; ok {
		t.Fatal("coal plant should be dropped")
	}
}

func TestPlantsFromTablesMissingColumn(t *testing.T) {
	plantRows := [][]string{
		{"Plant Code", "Plant Name"},
		{"1", "X"},
	}
	genRows := [][]string{
		{"Plant Code", "Energy Source 1"},
		{"1", "NUC"},
	}
	if _, err := plantsFromTables(plantRows, genRows); err == nil {
		t.Fatal("expected missing column error")
	}
}

func TestPlantsFromTablesMissingNuclearPlant(t *testing.T) {
	plantRows := [][]string{
		{"Utility ID", "Utility Name", "Plant Code", "Plant Name", "Street Address", "City", "State", "Zip", "County", "Latitude", "Longitude"},
		{"1", "A", "100", "Coal", "1 Main", "Town", "TX", "75001", "Dallas", "32.0", "-96.0"},
	}
	genRows := [][]string{
		{"Plant Code", "Energy Source 1"},
		{"6099", "NUC"},
	}
	if _, err := plantsFromTables(plantRows, genRows); err == nil {
		t.Fatal("expected missing nuclear plant error")
	}
}

func TestNormalizeZip(t *testing.T) {
	if got := normalizeZip("6385"); got != "06385" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeZip("93424"); got != "93424" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeZip(""); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeZip("not-a-zip"); got != "not-a-zip" {
		t.Fatalf("got %q", got)
	}
}

func TestPlantFromRowBadCoords(t *testing.T) {
	cols := plantCols{name: 0, address: 1, city: 2, state: 3, zip: 4, lat: 5, lon: 6}
	row := []string{"X", "a", "c", "TX", "1", "91", "0"}
	if _, err := plantFromRow(row, cols, "1"); err == nil {
		t.Fatal("expected coord error")
	}
}

func TestSplitHeaderEmpty(t *testing.T) {
	if _, _, err := splitHeader(nil); err == nil {
		t.Fatal("expected empty sheet")
	}
}

func TestDuplicatePlantCode(t *testing.T) {
	plantRows := [][]string{
		{"Utility ID", "Utility Name", "Plant Code", "Plant Name", "Street Address", "City", "State", "Zip", "County", "Latitude", "Longitude"},
		{"1", "A", "6099", "Diablo Canyon", "Avila Rd", "Avila Beach", "CA", "93424", "SLO", "35.2", "-120.8"},
		{"1", "A", "6099", "Diablo Canyon", "Avila Rd", "Avila Beach", "CA", "93424", "SLO", "35.2", "-120.8"},
	}
	genRows := [][]string{
		{"Plant Code", "Energy Source 1"},
		{"6099", "NUC"},
	}
	if _, err := plantsFromTables(plantRows, genRows); err == nil {
		t.Fatal("expected duplicate plant code")
	}
}
