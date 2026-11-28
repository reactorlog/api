package nrc

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestStoreCommitAndRollback(t *testing.T) {
	if !strings.Contains(advisoryLockSQL, "pg_advisory_xact_lock") {
		t.Fatal("import does not take the advisory lock")
	}
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("fail=%t", fail), func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectBegin()
			mock.ExpectExec(advisoryLockSQL).WillReturnResult(sqlmock.NewResult(0, 1))
			reactors := []Reactor{{
				Name: "Arkansas Nuclear 1", UnitNumber: "1", DocketNumber: "05000313",
				ReactorType: "PWR", SiteName: "Arkansas Nuclear",
				Location: "6 miles WNW of Russellville, AR", State: "AR",
			}}
			args := importArgs(reactors)
			exec := mock.ExpectExec(importSQL).WithArgs(args[0], args[1], args[2], args[3], args[4], args[5], args[6])
			if fail {
				exec.WillReturnError(fmt.Errorf("database failure"))
				mock.ExpectRollback()
			} else {
				exec.WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			}
			count, err := store(context.Background(), db, reactors)
			if fail && (err == nil || count != 0) {
				t.Fatalf("count = %d, error = %v", count, err)
			}
			if !fail && (err != nil || count != 1) {
				t.Fatalf("count = %d, error = %v", count, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTextArrayValue(t *testing.T) {
	value, err := textArray{`6 miles WNW of Russellville, AR`, `say "hello"`, `a\b`}.Value()
	if err != nil {
		t.Fatal(err)
	}
	want := `{"6 miles WNW of Russellville, AR","say \"hello\"","a\\b"}`
	if value != want {
		t.Fatalf("value = %s", value)
	}
}

func TestStoreEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	count, err := store(context.Background(), db, nil)
	if err != nil || count != 0 {
		t.Fatalf("count = %d, error = %v", count, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
