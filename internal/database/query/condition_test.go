package query

import (
	"reflect"
	"testing"

	"github.com/reactorlog/api/internal/database/expr"
)

func TestConditionValuesAreBound(t *testing.T) {
	cases := []struct {
		name      string
		condition expr.Condition
		where     string
		value     any
	}{
		{"string", expr.Column("name").Equals("Ada"), "name = $1", "Ada"},
		{"integer", expr.Column("n").GreaterThan(1), "n > $1", 1},
		{"float", expr.Column("latitude").GreaterThanOrEqual(-90.25), "latitude >= $1", -90.25},
		{"boolean", expr.Column("active").Equals(true), "active = $1", true},
		{"nil", expr.Column("deleted_at").Equals(nil), "deleted_at = $1", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sql, args, err := Select("id").From("sites").Where(tc.condition).Build()
			if err != nil {
				t.Fatal(err)
			}
			if sql != "SELECT id FROM sites WHERE "+tc.where {
				t.Fatalf("SQL = %q", sql)
			}
			if !reflect.DeepEqual(args, []any{tc.value}) {
				t.Fatalf("args = %#v", args)
			}
		})
	}
}

func TestNestedQueryValuesAreBound(t *testing.T) {
	value := "O'Reilly\\$3'; DROP TABLE sites; --"
	condition := expr.Or(
		expr.And(
			expr.Column("latitude").GreaterThanOrEqual(-90),
			expr.Column("latitude").LessThanOrEqual(90),
		),
		expr.Column("name").Equals(value),
	)
	sql, args, err := Select("id").From("sites").Where(condition).Build()
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT id FROM sites WHERE ((latitude >= $1 AND latitude <= $2) OR name = $3)"
	if sql != want {
		t.Fatalf("SQL = %q, want %q", sql, want)
	}
	if !reflect.DeepEqual(args, []any{-90, 90, value}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestQueryPlaceholderNumbering(t *testing.T) {
	b := &builder{}
	b.bind("seed")
	condition := expr.Or(
		expr.And(expr.Column("a").Equals(1), expr.Column("b").Equals(2)),
		expr.Column("c").Equals(3),
	)
	sql, err := b.buildCondition(condition)
	if err != nil {
		t.Fatal(err)
	}
	if sql != "((a = $2 AND b = $3) OR c = $4)" {
		t.Fatalf("SQL = %q", sql)
	}
	if !reflect.DeepEqual(b.args, []any{"seed", 1, 2, 3}) {
		t.Fatalf("args = %#v", b.args)
	}
}

func TestQueryConditionErrors(t *testing.T) {
	cases := []expr.Condition{
		expr.Column("").Equals(1),
		expr.And(nil, expr.Column("id").Equals(1)),
		expr.Or(expr.Column("id").Equals(1), nil),
	}
	for _, condition := range cases {
		sql, args, err := Select("id").From("sites").Where(condition).Build()
		if err == nil {
			t.Fatal("expected invalid condition error")
		}
		if sql != "" || args != nil {
			t.Fatalf("SQL = %q args = %#v", sql, args)
		}
	}
}
