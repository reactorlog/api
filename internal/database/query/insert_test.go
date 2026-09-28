package query

import "testing"

func TestInsertBuild(t *testing.T) {
	t.Run("table is required", func(t *testing.T) {
		_, _, err := Insert("").Columns("name").Values("Ada").Build()
		if err == nil || err.Error() != "insert table is required" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("columns are required", func(t *testing.T) {
		_, _, err := Insert("sites").Values("Ada").Build()
		if err == nil || err.Error() != "insert columns are required" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("values must match columns", func(t *testing.T) {
		_, _, err := Insert("sites").Columns("name").Values("Ada", "extra").Build()
		if err == nil || err.Error() != "insert values must match columns" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("insert", func(t *testing.T) {
		sql, args, err := Insert("sites").Columns("name", "id").Values("Ada", 7).Build()
		if err != nil {
			t.Fatal(err)
		}
		if sql != "INSERT INTO sites (name, id) VALUES ($1, $2)" {
			t.Fatalf("sql = %q", sql)
		}
		if len(args) != 2 || args[0] != "Ada" || args[1] != 7 {
			t.Fatalf("args = %#v", args)
		}
	})
}
