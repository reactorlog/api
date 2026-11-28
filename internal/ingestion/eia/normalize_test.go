package eia

import "testing"

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Diablo Canyon", "diablo canyon"},
		{"Diablo Canyon ", "diablo canyon"},
		{"diablo-canyon", "diablocanyon"},
		{"PSEG Salem Generating Station", "pseg salem generating station"},
		{"  Nine   Mile  Point  ", "nine mile point"},
		{"Browns Ferry", "browns ferry"},
	}
	for _, tc := range cases {
		if got := NormalizeName(tc.in); got != tc.want {
			t.Fatalf("NormalizeName(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
	if NormalizeName("Diablo Canyon") == NormalizeName("Diablo Canyon 2") {
		t.Fatal("distinct names collapsed")
	}
}
