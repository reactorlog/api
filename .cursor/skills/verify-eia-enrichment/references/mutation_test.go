package eia

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestVerificationEiaEnrichmentColumns(t *testing.T) {
	for _, word := range []string{
		"eia_plant_code",
		"address_line_1",
		"city",
		"state",
		"zip",
		"latitude",
		"longitude",
	} {
		if !strings.Contains(enrichSQL, word) {
			t.Fatalf("enrich omits %s", word)
		}
	}
	if !strings.Contains(enrichSQL, "site.name = incoming.site_name") {
		t.Fatal("enrich does not match by site name")
	}
	if !strings.Contains(enrichSQL, "is distinct from") {
		t.Fatal("enrich lacks is distinct from skip")
	}
	for _, word := range []string{"timezone", "location", "design", "phone", "email", "address_line_2", "capacity_mw"} {
		if strings.Contains(enrichSQL, word) {
			t.Fatalf("enrich writes protected or out-of-scope column %s", word)
		}
	}

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	plant := Plant{
		Code: "6099", Name: "Diablo Canyon", MatchKey: NormalizeName("Diablo Canyon"),
		AddressLine1: "1 Main", City: "Town", State: "CA", Zip: "93400",
		Latitude: 35, Longitude: -121,
	}
	expectCapacityImport(mock, plant, "1", 1138, 1, 1)

	result, err := store(context.Background(), db, []Plant{plant}, map[string][]Generator{
		"6099": {{PlantCode: "6099", ID: "1", NameplateMW: 1138}},
	})
	if err != nil || result.Matched != 1 || result.Updated != 1 || result.ReactorsUpdated != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
