package migrations

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSitesCreateSQL(t *testing.T) {
	sql, err := sitesCreateSQL()
	if err != nil {
		t.Fatal(err)
	}
	want := "CREATE TABLE sites (id UUID PRIMARY KEY DEFAULT uuidv7(), name TEXT NOT NULL, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)"
	if sql != want {
		t.Fatalf("sql = %q", sql)
	}
}

func TestSitesDropSQL(t *testing.T) {
	sql, err := sitesDropSQL()
	if err != nil {
		t.Fatal(err)
	}
	if sql != "DROP TABLE sites" {
		t.Fatalf("sql = %q", sql)
	}
}

func TestCreateSites(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	query, err := sitesCreateSQL()
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := createSites(context.Background(), tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDropSites(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	query, err := sitesDropSQL()
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := dropSites(context.Background(), tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
