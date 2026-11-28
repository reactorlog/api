package eia

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func samplePlant(code, name, site string) Plant {
	return Plant{
		Code: code, Name: name, MatchKey: NormalizeName(name),
		AddressLine1: "1 Main", City: "Town", State: "TX", Zip: "75001",
		Latitude: 32, Longitude: -96,
	}
}

func TestStoreSuccess(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"name"}).AddRow("Diablo Canyon")
	mock.ExpectQuery(`select name from sites order by name`).WillReturnRows(rows)
	mock.ExpectBegin()
	mock.ExpectExec(advisoryLockSQL).WillReturnResult(sqlmock.NewResult(0, 1))
	plant := samplePlant("6099", "Diablo Canyon", "Diablo Canyon")
	args := enrichArgs([]MatchedPlant{{SiteName: "Diablo Canyon", Plant: plant, Method: MatchExact}})
	mock.ExpectExec(enrichSQL).WithArgs(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7]).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := store(context.Background(), db, []Plant{plant})
	if err != nil || result.Matched != 1 || result.Updated != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreUnmatchedSite(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows := sqlmock.NewRows([]string{"name"}).AddRow("Missing")
	mock.ExpectQuery(`select name from sites order by name`).WillReturnRows(rows)
	_, err = store(context.Background(), db, []Plant{samplePlant("1", "Other", "Other")})
	if err == nil || !strings.Contains(err.Error(), "unmatched sites") {
		t.Fatal(err)
	}
}

func TestStoreBeginError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows := sqlmock.NewRows([]string{"name"}).AddRow("Diablo Canyon")
	mock.ExpectQuery(`select name from sites order by name`).WillReturnRows(rows)
	mock.ExpectBegin().WillReturnError(fmt.Errorf("begin failed"))
	_, err = store(context.Background(), db, []Plant{samplePlant("6099", "Diablo Canyon", "Diablo Canyon")})
	if err == nil || !strings.Contains(err.Error(), "begin EIA import") {
		t.Fatal(err)
	}
}

func TestImportFetchError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Import(ctx, nil)
	if err == nil {
		t.Fatal("expected fetch error")
	}
}

func TestTextAndFloatArrays(t *testing.T) {
	got, err := textArray{`say "hi"`, `a\b`}.Value()
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"say \"hi\"","a\\b"}` {
		t.Fatalf("%v", got)
	}
	f, err := floatArray{1.5, -2}.Value()
	if err != nil {
		t.Fatal(err)
	}
	if f != `{1.5,-2}` {
		t.Fatalf("%v", f)
	}
}

func TestListSitesEmpty(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(`select name from sites order by name`).WillReturnRows(sqlmock.NewRows([]string{"name"}))
	_, err = listSites(context.Background(), db)
	if err == nil || !strings.Contains(err.Error(), "no sites") {
		t.Fatal(err)
	}
}

func TestWriteEnrichmentEmpty(t *testing.T) {
	n, err := writeEnrichment(context.Background(), nil, nil)
	if err != nil || n != 0 {
		t.Fatalf("%d %v", n, err)
	}
}
