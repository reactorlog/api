package query

import (
	"strings"
	"testing"
)

func TestBuildConditionComparison(t *testing.T) {
	sql, args, err := buildCondition(Column("age").GreaterThan(18), 1)
	if err != nil {
		t.Fatal(err)
	}
	if sql != "age > $1" {
		t.Fatalf("sql = %q", sql)
	}
	if len(args) != 1 || args[0] != 18 {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildConditionLogical(t *testing.T) {
	condition := Column("age").GreaterThan(18).And(Column("name").Equals("Ada"))
	sql, args, err := buildCondition(condition, 1)
	if err != nil {
		t.Fatal(err)
	}
	if sql != "(age > $1 AND name = $2)" {
		t.Fatalf("sql = %q", sql)
	}
	if len(args) != 2 || args[0] != 18 || args[1] != "Ada" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildConditionOr(t *testing.T) {
	condition := Column("status").Equals("open").Or(Column("status").Equals("pending"))
	sql, args, err := buildCondition(condition, 2)
	if err != nil {
		t.Fatal(err)
	}
	if sql != "(status = $2 OR status = $3)" {
		t.Fatalf("sql = %q", sql)
	}
	if len(args) != 2 {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildConditionEmpty(t *testing.T) {
	_, _, err := buildCondition(Condition{}, 1)
	if err == nil || err.Error() != "condition is empty" {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildConditionBothKinds(t *testing.T) {
	condition := Condition{
		comparison: &Comparison{column: "id", operator: Operator("="), value: 1},
		logical: &Logical{
			left:     Column("id").Equals(1),
			operator: And,
			right:    Column("id").Equals(2),
		},
	}
	_, _, err := buildCondition(condition, 1)
	if err == nil || !strings.Contains(err.Error(), "both comparison and logical") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildConditionNestedError(t *testing.T) {
	condition := Condition{
		logical: &Logical{
			left:     Condition{},
			operator: And,
			right:    Column("id").Equals(1),
		},
	}
	_, _, err := buildCondition(condition, 1)
	if err == nil || err.Error() != "condition is empty" {
		t.Fatalf("error = %v", err)
	}

	condition = Condition{
		logical: &Logical{
			left:     Column("id").Equals(1),
			operator: Or,
			right:    Condition{},
		},
	}
	_, _, err = buildCondition(condition, 1)
	if err == nil || err.Error() != "condition is empty" {
		t.Fatalf("error = %v", err)
	}
}

func TestConditionBuilders(t *testing.T) {
	cases := []struct {
		name      string
		condition Condition
		wantSQL   string
		wantArg   any
	}{
		{"equals", Column("id").Equals(1), "id = $1", 1},
		{"not equals", Column("id").NotEquals(1), "id != $1", 1},
		{"greater than", Column("n").GreaterThan(2), "n > $1", 2},
		{"greater than or equal", Column("n").GreaterThanOrEqual(2), "n >= $1", 2},
		{"less than", Column("n").LessThan(2), "n < $1", 2},
		{"less than or equal", Column("n").LessThanOrEqual(2), "n <= $1", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sql, args, err := buildCondition(tc.condition, 1)
			if err != nil {
				t.Fatal(err)
			}
			if sql != tc.wantSQL {
				t.Fatalf("sql = %q, want %q", sql, tc.wantSQL)
			}
			if len(args) != 1 || args[0] != tc.wantArg {
				t.Fatalf("args = %#v", args)
			}
		})
	}
}
