package migrations

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMigrationsTableSQL(t *testing.T) {
	sql, err := migrationsTableSQL()
	if err != nil {
		t.Fatal(err)
	}
	want := "CREATE TABLE migrations (id UUID PRIMARY KEY DEFAULT uuidv7(), version INT NOT NULL UNIQUE, name TEXT NOT NULL, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)"
	if sql != want {
		t.Fatalf("sql = %q", sql)
	}
}

func TestAppliedVersionsQuery(t *testing.T) {
	sql, args, err := appliedVersionsQuery()
	if err != nil {
		t.Fatal(err)
	}
	if sql != "SELECT version FROM migrations" {
		t.Fatalf("sql = %q", sql)
	}
	if args != nil {
		t.Fatalf("args = %#v", args)
	}
}

func TestEnsureMigrationsTable(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	query, err := migrationsTableSQL()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(0, 0))

	if err := ensureMigrationsTable(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureMigrationsTableExecError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	query, err := migrationsTableSQL()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(query).WillReturnError(errors.New("exec failed"))

	err = ensureMigrationsTable(context.Background(), db)
	if err == nil || err.Error() != "create migrations table: exec failed" {
		t.Fatalf("error = %v", err)
	}
}

func TestScanVersions(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rows := sqlmock.NewRows([]string{"version"}).AddRow(1).AddRow(3)
	mock.ExpectQuery("SELECT version FROM migrations").WillReturnRows(rows)

	queryRows, err := db.Query("SELECT version FROM migrations")
	if err != nil {
		t.Fatal(err)
	}
	defer queryRows.Close()

	versions, err := scanVersions(queryRows)
	if err != nil {
		t.Fatal(err)
	}
	if !versions[1] || !versions[3] || versions[2] {
		t.Fatalf("versions = %#v", versions)
	}
}

func TestScanVersionsScanError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rows := sqlmock.NewRows([]string{"version"}).AddRow("bad")
	mock.ExpectQuery("SELECT version FROM migrations").WillReturnRows(rows)

	queryRows, err := db.Query("SELECT version FROM migrations")
	if err != nil {
		t.Fatal(err)
	}
	defer queryRows.Close()

	_, err = scanVersions(queryRows)
	if err == nil {
		t.Fatal("expected scan error")
	}
}

func TestAppliedVersions(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rows := sqlmock.NewRows([]string{"version"}).AddRow(1)
	mock.ExpectQuery("SELECT version FROM migrations").WillReturnRows(rows)

	versions, err := appliedVersions(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if !versions[1] {
		t.Fatalf("versions = %#v", versions)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAppliedVersionsQueryError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery("SELECT version FROM migrations").WillReturnError(errors.New("query failed"))

	_, err = appliedVersions(context.Background(), db)
	if err == nil || err.Error() != "query applied versions: query failed" {
		t.Fatalf("error = %v", err)
	}
}

func TestRunMigration(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectCommit()

	called := false
	err = runMigration(context.Background(), db, Migration{
		Version: 1,
		Name:    "test",
		Up: func(ctx context.Context, tx *sql.Tx) error {
			called = true
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("up not called")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunMigrationBeginError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	err = runMigration(context.Background(), db, Migration{Name: "test", Up: func(context.Context, *sql.Tx) error { return nil }})
	if err == nil || err.Error() != "begin transaction: begin failed" {
		t.Fatalf("error = %v", err)
	}
}

func TestCommitMigrationUpError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectRollback()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	err = commitMigration(context.Background(), tx, Migration{
		Up: func(context.Context, *sql.Tx) error { return errors.New("up failed") },
	})
	if err == nil || err.Error() != "up migration: up failed" {
		t.Fatalf("error = %v", err)
	}
}

func TestCommitMigrationCommitError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
	mock.ExpectRollback()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	err = commitMigration(context.Background(), tx, Migration{
		Up: func(context.Context, *sql.Tx) error { return nil },
	})
	if err == nil || err.Error() != "commit transaction: commit failed" {
		t.Fatalf("error = %v", err)
	}
}

func TestApplyPending(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectCommit()

	ran := []int{}
	err = applyPending(context.Background(), db, []Migration{
		{Version: 1, Name: "one", Up: func(context.Context, *sql.Tx) error { ran = append(ran, 1); return nil }},
		{Version: 2, Name: "two", Up: func(context.Context, *sql.Tx) error { ran = append(ran, 2); return nil }},
	}, map[int]bool{1: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(ran) != 1 || ran[0] != 2 {
		t.Fatalf("ran = %#v", ran)
	}
}

func TestApplyPendingError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	err = applyPending(context.Background(), db, []Migration{
		{Version: 1, Name: "one", Up: func(context.Context, *sql.Tx) error { return nil }},
	}, map[int]bool{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	createSQL, err := migrationsTableSQL()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(createSQL).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT version FROM migrations").
		WillReturnRows(sqlmock.NewRows([]string{"version"}))
	mock.ExpectBegin()
	mock.ExpectCommit()

	ran := false
	err = Run(context.Background(), db, []Migration{
		{Version: 1, Name: "one", Up: func(context.Context, *sql.Tx) error { ran = true; return nil }},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("migration not run")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunEnsureError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	createSQL, err := migrationsTableSQL()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(createSQL).WillReturnError(errors.New("exec failed"))

	err = Run(context.Background(), db, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunAppliedVersionsError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	createSQL, err := migrationsTableSQL()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(createSQL).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT version FROM migrations").WillReturnError(errors.New("query failed"))

	err = Run(context.Background(), db, nil)
	if err == nil || err.Error() != "query applied versions: query failed" {
		t.Fatalf("error = %v", err)
	}
}
