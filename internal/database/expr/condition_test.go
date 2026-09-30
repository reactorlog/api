package expr

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

type bindingFormatter struct {
	args []any
}

func (b *bindingFormatter) format(value any) (string, error) {
	b.args = append(b.args, value)
	return fmt.Sprintf("$%d", len(b.args)), nil
}

func TestComparisonBuild(t *testing.T) {
	cases := []struct {
		name string
		cond Condition
		sql  string
		arg  any
	}{
		{"equal", Column("name").Equals("Ada"), "name = $1", "Ada"},
		{"not equal", Column("name").NotEquals("Ada"), "name != $1", "Ada"},
		{"greater than", Column("n").GreaterThan(1), "n > $1", 1},
		{"greater than or equal", Column("n").GreaterThanOrEqual(1), "n >= $1", 1},
		{"less than", Column("n").LessThan(1), "n < $1", 1},
		{"less than or equal", Column("n").LessThanOrEqual(1), "n <= $1", 1},
		{"nil value still formats", Column("deleted_at").Equals(nil), "deleted_at = $1", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &bindingFormatter{}
			sql, err := Build(tc.cond, b.format)
			if err != nil {
				t.Fatal(err)
			}
			if sql != tc.sql {
				t.Fatalf("sql = %q", sql)
			}
			if !reflect.DeepEqual(b.args, []any{tc.arg}) {
				t.Fatalf("args = %#v", b.args)
			}
		})
	}
}

func TestComparisonContinuesPlaceholderNumbering(t *testing.T) {
	b := &bindingFormatter{args: []any{"seed", "seed"}}
	sql, err := Build(Column("count").Equals(0), b.format)
	if err != nil {
		t.Fatal(err)
	}
	if sql != "count = $3" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(b.args, []any{"seed", "seed", 0}) {
		t.Fatalf("args = %#v", b.args)
	}
}

func TestComparisonValidation(t *testing.T) {
	cases := []struct {
		name string
		cond Condition
		err  string
	}{
		{"empty column", Column("").Equals(1), "comparison column is required"},
		{"zero value", Comparison{}, "comparison column is required"},
		{"empty operator", Comparison{column: "id"}, "invalid comparison operator"},
		{"unsupported operator", Comparison{column: "id", operator: "LIKE"}, "invalid comparison operator"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &bindingFormatter{}
			sql, err := Build(tc.cond, b.format)
			if err == nil || err.Error() != tc.err {
				t.Fatalf("error = %v", err)
			}
			if sql != "" || len(b.args) != 0 {
				t.Fatalf("sql = %q args = %#v", sql, b.args)
			}
		})
	}
}

func TestComparisonPropagatesFormatterError(t *testing.T) {
	wantErr := errors.New("unsupported literal")
	formatter := func(value any) (string, error) {
		if value != "Ada" {
			t.Fatalf("value = %#v", value)
		}
		return "discarded", wantErr
	}
	sql, err := Build(Column("name").Equals("Ada"), formatter)
	if !errors.Is(err, wantErr) || sql != "" {
		t.Fatalf("sql = %q error = %v", sql, err)
	}
}

func TestBuildWithLiteralFormatter(t *testing.T) {
	condition := And(
		Column("latitude").GreaterThanOrEqual(-90),
		Column("latitude").LessThanOrEqual(90),
	)
	formatter := func(value any) (string, error) {
		return fmt.Sprint(value), nil
	}
	sql, err := Build(condition, formatter)
	if err != nil {
		t.Fatal(err)
	}
	if sql != "(latitude >= -90 AND latitude <= 90)" {
		t.Fatalf("sql = %q", sql)
	}
}

func TestBuildValidation(t *testing.T) {
	cases := []struct {
		name      string
		condition Condition
		formatter ValueFormatter
		err       string
	}{
		{"nil condition", nil, (&bindingFormatter{}).format, "condition is required"},
		{"nil formatter", Column("id").Equals(1), nil, "value formatter is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sql, err := Build(tc.condition, tc.formatter)
			if err == nil || err.Error() != tc.err {
				t.Fatalf("error = %v", err)
			}
			if sql != "" {
				t.Fatalf("sql = %q", sql)
			}
		})
	}
}

func TestLogicalBuild(t *testing.T) {
	b := &bindingFormatter{args: []any{0, 0, 0}}
	condition := Or(
		And(Column("a").Equals(1), Column("b").Equals(2)),
		Column("c").Equals(3),
	)
	sql, err := Build(condition, b.format)
	if err != nil {
		t.Fatal(err)
	}
	if sql != "((a = $4 AND b = $5) OR c = $6)" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(b.args, []any{0, 0, 0, 1, 2, 3}) {
		t.Fatalf("args = %#v", b.args)
	}
}

func TestLogicalUnformattedConditionDoesNotConsumePlaceholder(t *testing.T) {
	b := &bindingFormatter{args: []any{0, 1, 2, 3, 4, 5, 6}}
	left := &staticCondition{sql: "TRUE"}
	sql, err := Build(And(left, Column("id").Equals(8)), b.format)
	if err != nil {
		t.Fatal(err)
	}
	if sql != "(TRUE AND id = $8)" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(b.args, []any{0, 1, 2, 3, 4, 5, 6, 8}) {
		t.Fatalf("args = %#v", b.args)
	}
}

func TestLogicalLeftErrorSkipsRight(t *testing.T) {
	wantErr := errors.New("left")
	left := &staticCondition{err: wantErr}
	right := &staticCondition{sql: "right"}
	sql, err := Build(And(left, right), (&bindingFormatter{}).format)
	if !errors.Is(err, wantErr) || sql != "" {
		t.Fatalf("sql = %q error = %v", sql, err)
	}
	if right.calls != 0 {
		t.Fatalf("right calls = %d", right.calls)
	}
}

func TestLogicalRightErrorDiscardsSQL(t *testing.T) {
	wantErr := errors.New("right")
	b := &bindingFormatter{args: []any{"seed"}}
	right := &staticCondition{err: wantErr}
	sql, err := Build(Or(Column("a").Equals("kept-until-error"), right), b.format)
	if !errors.Is(err, wantErr) || sql != "" {
		t.Fatalf("sql = %q error = %v", sql, err)
	}
	if !reflect.DeepEqual(b.args, []any{"seed", "kept-until-error"}) {
		t.Fatalf("args = %#v", b.args)
	}
}

func TestLogicalValidation(t *testing.T) {
	cases := []struct {
		name string
		cond Condition
		err  string
	}{
		{"nil left", And(nil, Column("id").Equals(1)), "condition is required"},
		{"nil right", Or(Column("id").Equals(1), nil), "condition is required"},
		{"zero value", Logical{}, "invalid logical operator"},
		{"unsupported operator", Logical{operator: "XOR"}, "invalid logical operator"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sql, err := Build(tc.cond, (&bindingFormatter{}).format)
			if err == nil || err.Error() != tc.err {
				t.Fatalf("error = %v", err)
			}
			if sql != "" {
				t.Fatalf("sql = %q", sql)
			}
		})
	}
}

// staticCondition exercises logical branches without formatting a value.
type staticCondition struct {
	sql   string
	err   error
	calls int
}

func (c *staticCondition) build(ValueFormatter) (string, error) {
	c.calls++
	return c.sql, c.err
}
