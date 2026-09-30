package migrations

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

const migrationRecordSQL = "INSERT INTO migrations (version, name) VALUES ($1, $2)"

func newDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func noop(context.Context, *sql.Tx) error { return nil }

func validMigration(version int, name string) Migration {
	return Migration{Version: version, Name: name, Up: noop, Down: noop}
}

func TestMigrationsTableSQL(t *testing.T) {
	sql, err := migrationsTableSQL()
	if err != nil {
		t.Fatal(err)
	}
	want := "CREATE TABLE IF NOT EXISTS migrations (id UUID PRIMARY KEY DEFAULT uuidv7(), version INT NOT NULL UNIQUE, name TEXT NOT NULL, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)"
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

func TestValidateMigrationEntry(t *testing.T) {
	cases := []struct {
		name      string
		migration Migration
		want      string
	}{
		{
			name:      "negative version",
			migration: Migration{Version: -1, Name: "n", Up: noop, Down: noop},
			want:      "migration 2 version must be positive",
		},
		{
			name:      "zero version",
			migration: Migration{Version: 0, Name: "n", Up: noop, Down: noop},
			want:      "migration 2 version must be positive",
		},
		{
			name:      "missing name",
			migration: Migration{Version: 1, Up: noop, Down: noop},
			want:      "migration 2: name is required",
		},
		{
			name:      "missing up",
			migration: Migration{Version: 1, Name: "n", Down: noop},
			want:      "migration 2: up function is required",
		},
		{
			name:      "missing down",
			migration: Migration{Version: 1, Name: "n", Up: noop},
			want:      "migration 2: down function is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMigrationEntry(2, tc.migration)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error = %v", err)
			}
		})
	}

	t.Run("valid", func(t *testing.T) {
		if err := validateMigrationEntry(0, validMigration(1, "create_sites")); err != nil {
			t.Fatal(err)
		}
	})
}

func TestValidateMigrations(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if err := validateMigrations(nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("unique", func(t *testing.T) {
		err := validateMigrations([]Migration{
			validMigration(1, "one"),
			validMigration(2, "two"),
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("duplicate version", func(t *testing.T) {
		err := validateMigrations([]Migration{
			validMigration(1, "one"),
			validMigration(1, "again"),
		})
		if err == nil || err.Error() != "migration 1: version 1 already exists" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("entry error keeps its index", func(t *testing.T) {
		err := validateMigrations([]Migration{
			validMigration(1, "one"),
			{Version: 2, Up: noop, Down: noop},
		})
		if err == nil || err.Error() != "migration 1: name is required" {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestEnsureMigrationsTable(t *testing.T) {
	db, mock := newDB(t)
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
	db, mock := newDB(t)
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
	db, mock := newDB(t)
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
	db, mock := newDB(t)
	rows := sqlmock.NewRows([]string{"version"}).AddRow("bad")
	mock.ExpectQuery("SELECT version FROM migrations").WillReturnRows(rows)

	queryRows, err := db.Query("SELECT version FROM migrations")
	if err != nil {
		t.Fatal(err)
	}
	defer queryRows.Close()

	_, err = scanVersions(queryRows)
	if err == nil || !strings.Contains(err.Error(), "scan applied version") {
		t.Fatalf("error = %v", err)
	}
}

func TestScanVersionsRowsError(t *testing.T) {
	db, mock := newDB(t)
	rows := sqlmock.NewRows([]string{"version"}).AddRow(1).RowError(0, errors.New("next failed"))
	mock.ExpectQuery("SELECT version FROM migrations").WillReturnRows(rows)

	queryRows, err := db.Query("SELECT version FROM migrations")
	if err != nil {
		t.Fatal(err)
	}
	defer queryRows.Close()

	_, err = scanVersions(queryRows)
	if err == nil || !strings.Contains(err.Error(), "rows error") {
		t.Fatalf("error = %v", err)
	}
}

func TestAppliedVersions(t *testing.T) {
	db, mock := newDB(t)
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
	db, mock := newDB(t)
	mock.ExpectQuery("SELECT version FROM migrations").WillReturnError(errors.New("query failed"))

	_, err := appliedVersions(context.Background(), db)
	if err == nil || err.Error() != "query applied versions: query failed" {
		t.Fatalf("error = %v", err)
	}
}

func TestRecordMigration(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(migrationRecordSQL).
		WithArgs(1, "create_sites").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	err = recordMigration(context.Background(), tx, validMigration(1, "create_sites"))
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordMigrationExecError(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(migrationRecordSQL).
		WithArgs(1, "create_sites").
		WillReturnError(errors.New("exec failed"))
	mock.ExpectRollback()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	err = recordMigration(context.Background(), tx, validMigration(1, "create_sites"))
	if err == nil || err.Error() != "record migration: exec failed" {
		t.Fatalf("error = %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
}

func TestRunMigration(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(migrationRecordSQL).
		WithArgs(1, "test").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	called := false
	migration := validMigration(1, "test")
	migration.Up = func(context.Context, *sql.Tx) error {
		called = true
		return nil
	}
	if err := runMigration(context.Background(), db, migration); err != nil {
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
	db, mock := newDB(t)
	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	err := runMigration(context.Background(), db, validMigration(1, "test"))
	if err == nil || err.Error() != "begin transaction: begin failed" {
		t.Fatalf("error = %v", err)
	}
}

func TestRunMigrationUpError(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	migration := validMigration(1, "test")
	migration.Up = func(context.Context, *sql.Tx) error {
		return errors.New("up failed")
	}
	err := runMigration(context.Background(), db, migration)
	if err == nil || err.Error() != "up migration: up failed" {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunMigrationRecordError(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(migrationRecordSQL).
		WithArgs(1, "test").
		WillReturnError(errors.New("exec failed"))
	mock.ExpectRollback()

	err := runMigration(context.Background(), db, validMigration(1, "test"))
	if err == nil || err.Error() != "record migration: record migration: exec failed" {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunMigrationCommitError(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(migrationRecordSQL).
		WithArgs(1, "test").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	err := runMigration(context.Background(), db, validMigration(1, "test"))
	if err == nil || err.Error() != "commit transaction: commit failed" {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyPending(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectBegin()
	mock.ExpectExec(migrationRecordSQL).
		WithArgs(2, "two").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	ran := []int{}
	migrations := []Migration{
		validMigration(1, "one"),
		validMigration(2, "two"),
	}
	migrations[0].Up = func(context.Context, *sql.Tx) error {
		ran = append(ran, 1)
		return nil
	}
	migrations[1].Up = func(context.Context, *sql.Tx) error {
		ran = append(ran, 2)
		return nil
	}

	results, err := applyPending(context.Background(), db, migrations, map[int]bool{1: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(ran) != 1 || ran[0] != 2 {
		t.Fatalf("ran = %#v", ran)
	}
	if len(results) != 2 {
		t.Fatalf("results = %#v", results)
	}
	if results[0].Version != 1 || results[0].Name != "one" || results[0].Applied || results[0].Duration != 0 {
		t.Fatalf("already applied = %#v", results[0])
	}
	if results[1].Version != 2 || results[1].Name != "two" || !results[1].Applied || results[1].Duration < 0 {
		t.Fatalf("applied = %#v", results[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyPendingError(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	_, err := applyPending(context.Background(), db, []Migration{
		validMigration(1, "one"),
	}, map[int]bool{})
	if err == nil || !strings.Contains(err.Error(), `apply pending migration 1 "one"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestRun(t *testing.T) {
	db, mock := newDB(t)
	createSQL, err := migrationsTableSQL()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(createSQL).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT version FROM migrations").
		WillReturnRows(sqlmock.NewRows([]string{"version"}))
	mock.ExpectBegin()
	mock.ExpectExec(migrationRecordSQL).
		WithArgs(1, "one").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	ran := false
	migration := validMigration(1, "one")
	migration.Up = func(context.Context, *sql.Tx) error {
		ran = true
		return nil
	}
	results, err := Run(context.Background(), db, []Migration{migration})
	if err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("migration not run")
	}
	if len(results) != 1 || results[0].Version != 1 || results[0].Name != "one" || !results[0].Applied {
		t.Fatalf("results = %#v", results)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunValidateError(t *testing.T) {
	db, _ := newDB(t)
	_, err := Run(context.Background(), db, []Migration{{Version: 0}})
	if err == nil || err.Error() != "validate migrations: migration 0 version must be positive" {
		t.Fatalf("error = %v", err)
	}
}

func TestRunEnsureError(t *testing.T) {
	db, mock := newDB(t)
	createSQL, err := migrationsTableSQL()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(createSQL).WillReturnError(errors.New("exec failed"))

	_, err = Run(context.Background(), db, nil)
	if err == nil || err.Error() != "ensure migrations table: create migrations table: exec failed" {
		t.Fatalf("error = %v", err)
	}
}

func TestRunAppliedVersionsError(t *testing.T) {
	db, mock := newDB(t)
	createSQL, err := migrationsTableSQL()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(createSQL).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT version FROM migrations").WillReturnError(errors.New("query failed"))

	_, err = Run(context.Background(), db, nil)
	if err == nil || err.Error() != "query applied versions: query failed" {
		t.Fatalf("error = %v", err)
	}
}
