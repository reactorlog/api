package query

import (
	"reflect"
	"testing"

	"github.com/reactorlog/api/internal/database/expr"
)

func TestConditionCompatibility(t *testing.T) {
	cases := []struct {
		name      string
		condition Condition
		where     string
		value     any
	}{
		{"equal", Column("name").Equals("Ada"), "name = $1", "Ada"},
		{"not equal", Column("name").NotEquals("Ada"), "name != $1", "Ada"},
		{"greater than", Column("n").GreaterThan(1), "n > $1", 1},
		{"greater than or equal", Column("n").GreaterThanOrEqual(1), "n >= $1", 1},
		{"less than", Column("n").LessThan(1), "n < $1", 1},
		{"less than or equal", Column("n").LessThanOrEqual(1), "n <= $1", 1},
		{"nil still binds", Column("deleted_at").Equals(nil), "deleted_at = $1", nil},
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

func TestSharedConditionsInQuery(t *testing.T) {
	condition := expr.Or(
		expr.And(
			expr.Column("latitude").GreaterThanOrEqual(-90),
			expr.Column("latitude").LessThanOrEqual(90),
		),
		expr.Column("name").Equals("O'Reilly\\$3"),
	)
	sql, args, err := Select("id").From("sites").Where(condition).Build()
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT id FROM sites WHERE ((latitude >= $1 AND latitude <= $2) OR name = $3)"
	if sql != want {
		t.Fatalf("SQL = %q, want %q", sql, want)
	}
	if !reflect.DeepEqual(args, []any{-90, 90, "O'Reilly\\$3"}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestQueryLogicalCompatibility(t *testing.T) {
	b := &builder{}
	b.bind("seed")
	condition := Or(
		And(Column("a").Equals(1), Column("b").Equals(2)),
		Column("c").Equals(3),
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
	cases := []Condition{
		Comparison{},
		Logical{},
		And(nil, Column("id").Equals(1)),
		Or(Column("id").Equals(1), nil),
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
