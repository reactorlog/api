package eia

import (
	"reflect"
	"testing"
)

func TestSoftTokens(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"Diablo Canyon", []string{"diablo", "canyon"}},
		{"PSEG Salem Generating Station", []string{"salem"}},
		{"Donald C Cook", []string{"cook"}},
		{"D.C. Cook", []string{"d", "c", "cook"}},
		{"Harris (NC)", []string{"harris"}},
		{"Saint Lucie", []string{"st", "lucie"}},
		{"LaSalle Generating Station", []string{"lasalle"}},
		{"La Salle", []string{"la", "salle"}},
	}
	for _, tc := range cases {
		got := SoftTokens(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("SoftTokens(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestSoftMatch(t *testing.T) {
	if !softMatch("La Salle", "LaSalle Generating Station") {
		t.Fatal("La Salle")
	}
	if !softMatch("D.C. Cook", "Donald C Cook") {
		t.Fatal("D.C. Cook")
	}
	if !softMatch("Arkansas Nuclear", "Arkansas Nuclear One") {
		t.Fatal("Arkansas Nuclear")
	}
	// Shearon Harris soft-matches Harris (NC) via token subset; the alias map
	// still wins because Match tries aliases before soft matching.
	if !softMatch("Shearon Harris", "Harris (NC)") {
		t.Fatal("Shearon Harris soft tokens")
	}
}
