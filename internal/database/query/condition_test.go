package query

import (
	"errors"
	"testing"
)

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
		{"nil value still binds", Column("deleted_at").Equals(nil), "deleted_at = $1", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &builder{}
			sql, err := tc.cond.build(b)
			if err != nil {
				t.Fatal(err)
			}
			if sql != tc.sql {
				t.Fatalf("sql = %q", sql)
			}
			if len(b.args) != 1 || b.args[0] != tc.arg {
				t.Fatalf("args = %#v", b.args)
			}
		})
	}

	t.Run("numbering continues from the builder", func(t *testing.T) {
		b := &builder{}
		b.bind("seed")
		b.bind("seed")

		sql, err := Column("count").Equals(0).build(b)
		if err != nil {
			t.Fatal(err)
		}
		if sql != "count = $3" {
			t.Fatalf("sql = %q", sql)
		}
		if len(b.args) != 3 || b.args[2] != 0 {
			t.Fatalf("args = %#v", b.args)
		}
	})

	t.Run("empty column", func(t *testing.T) {
		b := &builder{}
		sql, err := Column("").Equals(1).build(b)
		if err == nil || err.Error() != "comparison column is required" {
			t.Fatalf("error = %v", err)
		}
		if sql != "" || b.args != nil {
			t.Fatalf("sql = %q args = %#v", sql, b.args)
		}
	})
}

func TestLogicalBuild(t *testing.T) {
	t.Run("placeholder advances by bind count", func(t *testing.T) {
		b := &builder{}
		b.bind(0)
		b.bind(0)
		b.bind(0)

		sql, err := Or(
			And(Column("a").Equals(1), Column("b").Equals(2)),
			Column("c").Equals(3),
		).build(b)
		if err != nil {
			t.Fatal(err)
		}
		if sql != "((a = $4 AND b = $5) OR c = $6)" {
			t.Fatalf("sql = %q", sql)
		}
		if len(b.args) != 6 || b.args[3] != 1 || b.args[4] != 2 || b.args[5] != 3 {
			t.Fatalf("args = %#v", b.args)
		}
	})

	t.Run("condition that binds nothing does not consume a placeholder", func(t *testing.T) {
		b := &builder{}
		for i := 0; i < 7; i++ {
			b.bind(i)
		}
		left := &staticCondition{sql: "TRUE"}
		right := &staticCondition{sql: "id =", bind: true, value: 8}

		sql, err := And(left, right).build(b)
		if err != nil {
			t.Fatal(err)
		}
		if sql != "(TRUE AND id = $8)" {
			t.Fatalf("sql = %q", sql)
		}
		if right.argsBefore != 7 {
			t.Fatalf("right args before = %d", right.argsBefore)
		}
		if len(b.args) != 8 || b.args[7] != 8 {
			t.Fatalf("args = %#v", b.args)
		}
	})

	t.Run("left error skips right", func(t *testing.T) {
		left := &staticCondition{err: errors.New("left")}
		right := &staticCondition{sql: "right"}

		sql, err := And(left, right).build(&builder{})
		if err == nil || err.Error() != "left" {
			t.Fatalf("error = %v", err)
		}
		if sql != "" {
			t.Fatalf("sql = %q", sql)
		}
		if right.calls != 0 {
			t.Fatalf("right calls = %d", right.calls)
		}
	})

	t.Run("right error discards sql", func(t *testing.T) {
		b := &builder{}
		b.bind("seed")
		left := &staticCondition{sql: "a =", bind: true, value: "kept-until-error"}
		right := &staticCondition{err: errors.New("right")}

		sql, err := Or(left, right).build(b)
		if err == nil || err.Error() != "right" {
			t.Fatalf("error = %v", err)
		}
		if sql != "" {
			t.Fatalf("sql = %q", sql)
		}
		if left.argsBefore != 1 || right.argsBefore != 2 {
			t.Fatalf("args before left=%d right=%d", left.argsBefore, right.argsBefore)
		}
	})

	t.Run("nil child", func(t *testing.T) {
		cases := []Condition{
			And(nil, Column("id").Equals(1)),
			Or(Column("id").Equals(1), nil),
		}
		for _, cond := range cases {
			func() {
				defer func() {
					if recover() == nil {
						t.Fatal("expected panic")
					}
				}()
				_, _ = cond.build(&builder{})
			}()
		}
	})
}

// staticCondition is a Condition whose binds and error are chosen by the test.
// Comparison always binds one value and returns a nil error, so Logical's other
// branches are otherwise unreachable.
type staticCondition struct {
	sql        string
	value      any
	bind       bool
	err        error
	calls      int
	argsBefore int
}

func (c *staticCondition) build(b *builder) (string, error) {
	c.calls++
	c.argsBefore = len(b.args)
	if c.err != nil {
		return "", c.err
	}
	sql := c.sql
	if c.bind {
		sql += " " + b.bind(c.value)
	}
	return sql, nil
}
