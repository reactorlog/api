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
		{"x", "y", "Plant Code", "Plant Name", "Generator ID", "Energy Source 1", "Nameplate Capacity (MW)"},
		{"", "", "6099", "Diablo Canyon", "1", "NUC", "1138.5"},
		{"", "", "100", "Coal Plant", "1", "BIT", "500"},
		{"", "", "649", "Vogtle", "2", "NUC", "1102"},
		{"", "", "649", "Vogtle", "1.0", "NUC", "1101"},
	}
	plants, generators, err := plantsFromTables(plantRows, genRows)
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
	diabloGens := generators["6099"]
	if len(diabloGens) != 1 || diabloGens[0].ID != "1" || diabloGens[0].NameplateMW != 1138.5 {
		t.Fatalf("diablo generators %+v", diabloGens)
	}
	vogtle := generators["649"]
	if len(vogtle) != 2 || vogtle[0].ID != "2" || vogtle[0].NameplateMW != 1102 || vogtle[1].ID != "1" || vogtle[1].NameplateMW != 1101 {
		t.Fatalf("vogtle generators %+v", vogtle)
	}
	if _, ok := generators["100"]; ok {
		t.Fatal("coal generator should be dropped")
	}
}

func TestPlantsFromTablesMissingColumn(t *testing.T) {
	plantRows := [][]string{
		{"Plant Code", "Plant Name"},
		{"1", "X"},
	}
	genRows := [][]string{
		{"Plant Code", "Generator ID", "Energy Source 1", "Nameplate Capacity (MW)"},
		{"1", "1", "NUC", "100"},
	}
	if _, _, err := plantsFromTables(plantRows, genRows); err == nil {
		t.Fatal("expected missing column error")
	}
}

func TestPlantsFromTablesMissingNuclearPlant(t *testing.T) {
	plantRows := [][]string{
		{"Utility ID", "Utility Name", "Plant Code", "Plant Name", "Street Address", "City", "State", "Zip", "County", "Latitude", "Longitude"},
		{"1", "A", "100", "Coal", "1 Main", "Town", "TX", "75001", "Dallas", "32.0", "-96.0"},
	}
	genRows := [][]string{
		{"Plant Code", "Generator ID", "Energy Source 1", "Nameplate Capacity (MW)"},
		{"6099", "1", "NUC", "100"},
	}
	if _, _, err := plantsFromTables(plantRows, genRows); err == nil {
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
		{"Plant Code", "Generator ID", "Energy Source 1", "Nameplate Capacity (MW)"},
		{"6099", "1", "NUC", "100"},
	}
	if _, _, err := plantsFromTables(plantRows, genRows); err == nil {
		t.Fatal("expected duplicate plant code")
	}
}

func TestPlantsFromTablesMissingNameplate(t *testing.T) {
	plantRows := [][]string{
		{"Plant Code", "Plant Name", "Street Address", "City", "State", "Zip", "Latitude", "Longitude"},
		{"6099", "Diablo Canyon", "Avila Rd", "Avila Beach", "CA", "93424", "35.2", "-120.8"},
	}
	genRows := [][]string{
		{"Plant Code", "Generator ID", "Energy Source 1"},
		{"6099", "1", "NUC"},
	}
	if _, _, err := plantsFromTables(plantRows, genRows); err == nil {
		t.Fatal("expected missing nameplate column")
	}
}

func TestPlantsFromTablesMissingGeneratorID(t *testing.T) {
	plantRows := [][]string{
		{"Plant Code", "Plant Name", "Street Address", "City", "State", "Zip", "Latitude", "Longitude"},
		{"6099", "Diablo Canyon", "Avila Rd", "Avila Beach", "CA", "93424", "35.2", "-120.8"},
	}
	genRows := [][]string{
		{"Plant Code", "Generator ID", "Energy Source 1", "Nameplate Capacity (MW)"},
		{"6099", "", "NUC", "1138"},
	}
	if _, _, err := plantsFromTables(plantRows, genRows); err == nil {
		t.Fatal("expected missing generator id")
	}
}
