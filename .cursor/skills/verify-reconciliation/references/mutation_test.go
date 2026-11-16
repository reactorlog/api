package nrc

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"regexp"
	"testing"
)

func TestVerificationExistingSite(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("^" + regexp.QuoteMeta("SELECT site_id FROM reactors WHERE docket_number = $1") + "$").WithArgs("05000313").WillReturnRows(sqlmock.NewRows([]string{"site_id"}).AddRow("original-site"))
	mock.ExpectExec("UPDATE sites").WithArgs("renamed", "place", "AR", "original-site").WillReturnResult(sqlmock.NewResult(0, 1))
	id, err := siteFor(context.Background(), tx, Reactor{DocketNumber: "05000313", SiteName: "renamed", Location: "place", State: "AR"})
	if err != nil || id != "original-site" {
		t.Fatalf("site=%s err=%v", id, err)
	}
	mock.ExpectRollback()
	tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
