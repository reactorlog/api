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
	for _, word := range []string{"timezone", "location", "design", "phone", "email", "address_line_2"} {
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
	rows := sqlmock.NewRows([]string{"name"}).AddRow("Diablo Canyon")
	mock.ExpectQuery(`select name from sites order by name`).WillReturnRows(rows)
	mock.ExpectBegin()
	mock.ExpectExec(advisoryLockSQL).WillReturnResult(sqlmock.NewResult(0, 1))
	args := enrichArgs([]MatchedPlant{{SiteName: "Diablo Canyon", Plant: plant, Method: MatchExact}})
	mock.ExpectExec(enrichSQL).WithArgs(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7]).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := store(context.Background(), db, []Plant{plant})
	if err != nil || result.Matched != 1 || result.Updated != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
