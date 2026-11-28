package nrc

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestVerificationNewDocketJoinsSite(t *testing.T) {
	if strings.Contains(importSQL, "and false") {
		t.Fatal("name match is disabled")
	}
	if !strings.Contains(importSQL, "occupied.name = incoming.site_name") {
		t.Fatal("import does not match a site by name")
	}
	if strings.Contains(importSQL, "occupied.location = incoming.location") {
		t.Fatal("import still matches a site by location")
	}
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	reactors := []Reactor{{
		Name: "Arkansas Nuclear 1", UnitNumber: "1", DocketNumber: "05000313",
		ReactorType: "PWR", SiteName: "Arkansas Nuclear", Location: "Russellville, AR", State: "AR",
	}}
	mock.ExpectBegin()
	mock.ExpectExec(advisoryLockSQL).WillReturnResult(sqlmock.NewResult(0, 1))
	args := importArgs(reactors)
	mock.ExpectExec(importSQL).WithArgs(args[0], args[1], args[2], args[3], args[4], args[5], args[6]).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	count, err := store(context.Background(), db, reactors)
	if err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
