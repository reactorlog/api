package schema

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/reactorlog/api/internal/database/expr"
)

func TestBuildColumnChecks(t *testing.T) {
	column := expr.Column("value")
	cases := []struct {
		name   string
		column Column
		want   string
	}{
		{"equal", Int("value").Check(column.Equals(5)), "value INT CHECK (value = 5)"},
		{"not equal", Int("value").Check(column.NotEquals(5)), "value INT CHECK (value != 5)"},
		{"greater", Int("value").Check(column.GreaterThan(5)), "value INT CHECK (value > 5)"},
		{"greater or equal", Int("value").Check(column.GreaterThanOrEqual(5)), "value INT CHECK (value >= 5)"},
		{"less", Int("value").Check(column.LessThan(5)), "value INT CHECK (value < 5)"},
		{"less or equal", Int("value").Check(column.LessThanOrEqual(5)), "value INT CHECK (value <= 5)"},
		{"nullable bounds", Float("latitude").Between(-90, 90), "latitude FLOAT CHECK ((latitude >= -90 AND latitude <= 90))"},
		{"required bounds", Float("longitude").NotNull().Between(-180, 180), "longitude FLOAT NOT NULL CHECK ((longitude >= -180 AND longitude <= 180))"},
		{
			"composed condition",
			Int("value").Check(expr.Or(expr.And(column.GreaterThan(0), column.LessThan(10)), column.Equals(100))),
			"value INT CHECK (((value > 0 AND value < 10) OR value = 100))",
		},
		{
			"repeated checks",
			Int("value").Check(column.GreaterThan(0)).Check(column.LessThan(10)),
			"value INT CHECK (value > 0) CHECK (value < 10)",
		},
		{
			"other modifiers",
			Int("value").NotNull().Unique().Check(column.GreaterThan(0)),
			"value INT NOT NULL UNIQUE CHECK (value > 0)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertColumnSQL(t, tc.column, tc.want)
		})
	}
}

func assertColumnSQL(t *testing.T, column Column, want string) {
	t.Helper()
	got, err := buildColumn(column)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("SQL = %q, want %q", got, want)
	}
}

func TestCheckBranchesAreIndependent(t *testing.T) {
	value := expr.Column("value")
	base := Int("value").Check(value.GreaterThan(0)).Check(value.LessThan(100))
	// A spare backing-array slot makes accidental append aliasing observable.
	checks := make([]expr.Condition, len(base.checks), len(base.checks)+1)
	copy(checks, base.checks)
	base.checks = checks
	first := base.Check(value.NotEquals(10))
	second := base.Check(value.NotEquals(20))
	prefix := "value INT CHECK (value > 0) CHECK (value < 100)"
	assertColumnSQL(t, base, prefix)
	assertColumnSQL(t, first, prefix+" CHECK (value != 10)")
	assertColumnSQL(t, second, prefix+" CHECK (value != 20)")
}

func TestBuildRejectsInvalidChecks(t *testing.T) {
	cases := []struct {
		name      string
		condition expr.Condition
	}{
		{"nil", nil},
		{"missing column", expr.Column("").GreaterThan(0)},
		{"missing left", expr.And(nil, expr.Column("value").Equals(1))},
		{"missing right", expr.Or(expr.Column("value").Equals(1), nil)},
		{"unsupported value", expr.Column("value").Equals([]byte("value"))},
		{"nonfinite bound", expr.Column("value").LessThan(math.Inf(1))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Build(Table{Name: "example", Columns: []Column{Int("value").Check(tc.condition)}})
			if err == nil {
				t.Fatalf("expected error, got SQL %q", got)
			}
			if !strings.Contains(err.Error(), `build column "value": build check:`) {
				t.Fatalf("error lost column/check context: %v", err)
			}
		})
	}
}

func TestBuildTableWithBounds(t *testing.T) {
	got, err := Build(Table{
		Name: "sites",
		Columns: []Column{
			Float("latitude").Between(-90, 90),
			Float("longitude").Between(-180, 180),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "CREATE TABLE sites (latitude FLOAT CHECK ((latitude >= -90 AND latitude <= 90)), longitude FLOAT CHECK ((longitude >= -180 AND longitude <= 180)))"
	if got != want {
		t.Fatalf("SQL = %q, want %q", got, want)
	}
}

func TestGeneratedBoundsAreEnforced(t *testing.T) {
	tx := checkTestTransaction(t)
	statement, err := Build(Table{
		Name: "schema_check_bounds",
		Columns: []Column{
			Float("latitude").Between(-90, 90),
			Float("longitude").Between(-180, 180),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	execCheckSQL(t, tx, statement)
	cases := []struct {
		name      string
		latitude  any
		longitude any
		invalid   bool
	}{
		{"minimum boundaries", -90, -180, false},
		{"maximum boundaries", 90, 180, false},
		{"interior", 40.5, -73.8, false},
		{"both null", nil, nil, false},
		{"latitude null", nil, 100, false},
		{"longitude null", 50, nil, false},
		{"latitude too low", -90.01, 0, true},
		{"latitude too high", 90.01, 0, true},
		{"longitude too low", 0, -180.01, true},
		{"longitude too high", 0, 180.01, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertCheckInsert(t, tx, "INSERT INTO schema_check_bounds VALUES ($1, $2)", tc.invalid, tc.latitude, tc.longitude)
		})
	}
}

func TestGeneratedCheckStringsAreEscaped(t *testing.T) {
	tx := checkTestTransaction(t)
	value := "O'Brien\\path\\n\nend"
	for _, setting := range []string{"on", "off"} {
		t.Run(setting, func(t *testing.T) {
			execCheckSQL(t, tx, "SET LOCAL standard_conforming_strings = "+setting)
			table := "schema_check_strings_" + setting
			statement, err := Build(Table{
				Name:    table,
				Columns: []Column{Text("value").Check(expr.Column("value").Equals(value))},
			})
			if err != nil {
				t.Fatal(err)
			}
			execCheckSQL(t, tx, statement)
			insert := fmt.Sprintf("INSERT INTO %s VALUES ($1)", table)
			assertCheckInsert(t, tx, insert, false, value)
			assertCheckInsert(t, tx, insert, true, "different")
		})
	}
}

func checkTestTransaction(t *testing.T) pgx.Tx {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	return tx
}

func execCheckSQL(t *testing.T, tx pgx.Tx, statement string) {
	t.Helper()
	if _, err := tx.Exec(context.Background(), statement); err != nil {
		t.Fatalf("exec %s: %v", statement, err)
	}
}

func assertCheckInsert(t *testing.T, tx pgx.Tx, statement string, invalid bool, args ...any) {
	t.Helper()
	ctx := context.Background()
	// Each insert gets a savepoint so an expected violation cannot abort the test transaction.
	insert, err := tx.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = insert.Rollback(ctx) }()
	_, err = insert.Exec(ctx, statement, args...)
	if !invalid {
		if err != nil {
			t.Fatalf("valid insert: %v", err)
		}
		return
	}
	var violation *pgconn.PgError
	if !errors.As(err, &violation) {
		t.Fatalf("expected check violation, got %v", err)
	}
	if violation.Code != "23514" {
		t.Fatalf("expected check violation (23514), got %s: %v", violation.Code, err)
	}
}
