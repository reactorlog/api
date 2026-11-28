package eia

import "testing"

func TestMatchExact(t *testing.T) {
	sites := []string{"Diablo Canyon", "Vogtle"}
	plants := []Plant{
		{Code: "6099", Name: "Diablo Canyon", MatchKey: NormalizeName("Diablo Canyon")},
		{Code: "649", Name: "Vogtle", MatchKey: NormalizeName("Vogtle")},
	}
	result, err := Match(sites, plants)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Matched) != 2 || len(result.UnmatchedSites) != 0 {
		t.Fatalf("%+v", result)
	}
	for _, m := range result.Matched {
		if m.Method != MatchExact {
			t.Fatalf("%+v", m)
		}
	}
}

func TestMatchSoftMissList(t *testing.T) {
	sites := []string{
		"Arkansas Nuclear", "Braidwood", "D.C. Cook", "Davis-Besse", "Farley",
		"Hope Creek", "La Salle", "Saint Lucie", "Summer", "Waterford", "Shearon Harris",
	}
	plants := []Plant{
		plant("8055", "Arkansas Nuclear One"),
		plant("6022", "Braidwood Generation Station"),
		plant("6000", "Donald C Cook"),
		plant("6149", "Davis Besse"),
		plant("6001", "Joseph M Farley"),
		plant("6118", "PSEG Hope Creek Generating Station"),
		plant("6026", "LaSalle Generating Station"),
		plant("6045", "St Lucie"),
		plant("6127", "V C Summer"),
		plant("4270", "Waterford 3"),
		plant("6015", "Harris (NC)"),
		plant("8011", "Crane Clean Energy Center"),
	}
	result, err := Match(sites, plants)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.UnmatchedSites) != 0 {
		t.Fatalf("unmatched sites %v", result.UnmatchedSites)
	}
	if len(result.UnmatchedPlants) != 1 || result.UnmatchedPlants[0].Code != "8011" {
		t.Fatalf("expected only Crane unmatched, got %+v", result.UnmatchedPlants)
	}
	bySite := map[string]MatchedPlant{}
	for _, m := range result.Matched {
		bySite[m.SiteName] = m
	}
	if bySite["Shearon Harris"].Method != MatchAlias || bySite["Shearon Harris"].Plant.Code != "6015" {
		t.Fatalf("%+v", bySite["Shearon Harris"])
	}
	if bySite["D.C. Cook"].Method != MatchSoft || bySite["D.C. Cook"].Plant.Code != "6000" {
		t.Fatalf("%+v", bySite["D.C. Cook"])
	}
	if bySite["Diablo Canyon"].Plant.Code != "" {
		// not in this fixture
	}
}

func TestMatchUnmatchedSite(t *testing.T) {
	result, err := Match([]string{"Missing Site"}, []Plant{plant("1", "Other")})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.UnmatchedSites) != 1 || result.UnmatchedSites[0] != "Missing Site" {
		t.Fatalf("%+v", result)
	}
}

func TestMatchDuplicatePlantKey(t *testing.T) {
	_, err := Match([]string{"Diablo Canyon"}, []Plant{
		plant("1", "Diablo Canyon"),
		plant("2", "Diablo  Canyon"),
	})
	if err == nil {
		t.Fatal("expected duplicate plant error")
	}
}

func TestMatchAmbiguousSoft(t *testing.T) {
	_, err := Match([]string{"Cook"}, []Plant{
		plant("1", "Donald C Cook"),
		plant("2", "Cook Nuclear Plant"),
	})
	if err == nil {
		t.Fatal("expected ambiguous soft match")
	}
}

func plant(code, name string) Plant {
	return Plant{Code: code, Name: name, MatchKey: NormalizeName(name)}
}
