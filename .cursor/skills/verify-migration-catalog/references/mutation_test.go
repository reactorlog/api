package migrations

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestVerificationMigrationDirection(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	expectScript(mock, migrationsTableSQL)
	mock.ExpectQuery(appliedVersionsSQL).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(2))
	mock.ExpectBegin()
	expectScript(mock, createSitesUp)
	mock.ExpectExec(recordMigrationSQL).
		WithArgs(1, "one").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	first := Migration{Version: 1, Name: "one", Up: createSitesUp, Down: createSitesDown}
	second := Migration{Version: 2, Name: "two", Up: createReactorsUp, Down: createReactorsDown}
	results, err := Run(context.Background(), db, []Migration{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || !results[0].Applied || results[0].Version != 1 || results[1].Applied || results[1].Version != 2 {
		t.Fatalf("results = %#v", results)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
