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

	plant := samplePlant("6099", "Diablo Canyon", "Diablo Canyon")
	expectCapacityImport(mock, plant, "1", 1138.5, 1, 1)

	result, err := store(context.Background(), db, []Plant{plant}, diabloGenerators(1138.5))
	if err != nil || result.Matched != 1 || result.Updated != 1 || result.ReactorsUpdated != 1 {
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
	_, err = store(context.Background(), db, []Plant{samplePlant("1", "Other", "Other")}, nil)
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
	_, err = store(context.Background(), db, []Plant{samplePlant("6099", "Diablo Canyon", "Diablo Canyon")}, nil)
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

func TestWriteCapacityEmpty(t *testing.T) {
	n, err := writeCapacity(context.Background(), nil, nil)
	if err != nil || n != 0 {
		t.Fatalf("%d %v", n, err)
	}
}

func TestStoreGeneratorMismatch(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	plant := samplePlant("6099", "Diablo Canyon", "Diablo Canyon")
	mock.ExpectQuery(`select name from sites order by name`).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Diablo Canyon"))
	mock.ExpectBegin()
	mock.ExpectExec(advisoryLockSQL).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(reactorsForSitesSQL).WithArgs(textArray{"Diablo Canyon"}).WillReturnRows(
		sqlmock.NewRows([]string{"name", "unit_number"}).AddRow("Diablo Canyon", "2"),
	)
	mock.ExpectRollback()

	_, err = store(context.Background(), db, []Plant{plant}, diabloGenerators(1138.5))
	if err == nil || !strings.Contains(err.Error(), "Diablo Canyon") || !strings.Contains(err.Error(), "2") {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreCapacityIdempotent(t *testing.T) {
	if !strings.Contains(capacitySQL, "capacity_mw") || !strings.Contains(capacitySQL, "is distinct from") {
		t.Fatal("capacity update is not conditional")
	}
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	plant := samplePlant("6099", "Diablo Canyon", "Diablo Canyon")
	expectCapacityImport(mock, plant, "1", 1138.5, 1, 1)
	expectCapacityImport(mock, plant, "1", 1138.5, 0, 0)

	first, err := store(context.Background(), db, []Plant{plant}, diabloGenerators(1138.5))
	if err != nil || first.ReactorsUpdated != 1 {
		t.Fatalf("%+v %v", first, err)
	}
	second, err := store(context.Background(), db, []Plant{plant}, diabloGenerators(1138.5))
	if err != nil || second.Updated != 0 || second.ReactorsUpdated != 0 {
		t.Fatalf("%+v %v", second, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func diabloGenerators(mw float64) map[string][]Generator {
	return map[string][]Generator{
		"6099": {{PlantCode: "6099", ID: "1", NameplateMW: mw}},
	}
}

func expectCapacityImport(mock sqlmock.Sqlmock, plant Plant, unit string, mw float64, enrichN, capN int64) {
	mock.ExpectQuery(`select name from sites order by name`).WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow(plant.Name))
	mock.ExpectBegin()
	mock.ExpectExec(advisoryLockSQL).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(reactorsForSitesSQL).WithArgs(textArray{plant.Name}).WillReturnRows(
		sqlmock.NewRows([]string{"name", "unit_number"}).AddRow(plant.Name, unit),
	)
	args := enrichArgs([]MatchedPlant{{SiteName: plant.Name, Plant: plant, Method: MatchExact}})
	mock.ExpectExec(enrichSQL).WithArgs(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7]).WillReturnResult(sqlmock.NewResult(0, enrichN))
	capArgs := capacityArgs([]capacityRow{{SiteName: plant.Name, UnitNumber: unit, CapacityMW: mw}})
	mock.ExpectExec(capacitySQL).WithArgs(capArgs[0], capArgs[1], capArgs[2]).WillReturnResult(sqlmock.NewResult(0, capN))
	mock.ExpectCommit()
}
