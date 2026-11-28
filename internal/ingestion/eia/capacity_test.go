package eia

import (
	"strings"
	"testing"
)

func TestAssignmentsMatch(t *testing.T) {
	matched := []MatchedPlant{
		{SiteName: "Vogtle", Plant: Plant{Code: "649"}, Method: MatchExact},
		{SiteName: "Diablo Canyon", Plant: Plant{Code: "6099"}, Method: MatchExact},
	}
	units := []reactorUnit{
		{SiteName: "Vogtle", UnitNumber: "2"},
		{SiteName: "Diablo Canyon", UnitNumber: "1"},
		{SiteName: "Vogtle", UnitNumber: "1"},
	}
	generators := map[string][]Generator{
		"649": {
			{PlantCode: "649", ID: "2", NameplateMW: 1102},
			{PlantCode: "649", ID: "1", NameplateMW: 1101},
		},
		"6099": {{PlantCode: "6099", ID: "1", NameplateMW: 1138.5}},
	}
	rows, err := assignments(matched, units, generators)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].SiteName != "Diablo Canyon" || rows[0].CapacityMW != 1138.5 {
		t.Fatalf("%+v", rows)
	}
	if rows[1].SiteName != "Vogtle" || rows[1].UnitNumber != "1" || rows[1].CapacityMW != 1101 {
		t.Fatalf("%+v", rows)
	}
	if rows[2].UnitNumber != "2" || rows[2].CapacityMW != 1102 {
		t.Fatalf("%+v", rows)
	}
}

func TestAssignmentsMismatch(t *testing.T) {
	matched := []MatchedPlant{{SiteName: "Vogtle", Plant: Plant{Code: "649"}, Method: MatchExact}}
	units := []reactorUnit{{SiteName: "Vogtle", UnitNumber: "1"}}
	generators := map[string][]Generator{
		"649": {
			{ID: "1", NameplateMW: 1101},
			{ID: "2", NameplateMW: 1102},
		},
	}
	_, err := assignments(matched, units, generators)
	if err == nil || !strings.Contains(err.Error(), "Vogtle") || !strings.Contains(err.Error(), "2") {
		t.Fatal(err)
	}
}

func TestAssignmentsDuplicateIDs(t *testing.T) {
	matched := []MatchedPlant{{SiteName: "Vogtle", Plant: Plant{Code: "649"}, Method: MatchExact}}
	units := []reactorUnit{{SiteName: "Vogtle", UnitNumber: "1"}}
	generators := map[string][]Generator{
		"649": {
			{ID: "1", NameplateMW: 1101},
			{ID: "1", NameplateMW: 1102},
		},
	}
	_, err := assignments(matched, units, generators)
	if err == nil || !strings.Contains(err.Error(), "duplicate generator id 1") {
		t.Fatal(err)
	}

	units = []reactorUnit{
		{SiteName: "Vogtle", UnitNumber: "1"},
		{SiteName: "Vogtle", UnitNumber: "1"},
	}
	generators = map[string][]Generator{"649": {{ID: "1", NameplateMW: 1101}}}
	_, err = assignments(matched, units, generators)
	if err == nil || !strings.Contains(err.Error(), "duplicate unit number 1") {
		t.Fatal(err)
	}
}
