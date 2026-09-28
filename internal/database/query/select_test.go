package query

import "testing"

func TestSelectBuild(t *testing.T) {
	t.Run("columns are required", func(t *testing.T) {
		_, _, err := Select().From("sites").Build()
		if err == nil || err.Error() != "select columns are required" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("table is required", func(t *testing.T) {
		_, _, err := Select("id").Build()
		if err == nil || err.Error() != "select table is required" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("without where", func(t *testing.T) {
		sql, args, err := Select("id", "name").From("sites").Build()
		if err != nil {
			t.Fatal(err)
		}
		if sql != "SELECT id, name FROM sites" {
			t.Fatalf("sql = %q", sql)
		}
		if args != nil {
			t.Fatalf("args = %#v", args)
		}
	})

	t.Run("with where", func(t *testing.T) {
		sql, args, err := Select("id").From("sites").Where(Column("name").Equals("Ada")).Build()
		if err != nil {
			t.Fatal(err)
		}
		if sql != "SELECT id FROM sites WHERE name = $1" {
			t.Fatalf("sql = %q", sql)
		}
		if len(args) != 1 || args[0] != "Ada" {
			t.Fatalf("args = %#v", args)
		}
	})

	t.Run("where build error", func(t *testing.T) {
		_, _, err := Select("id").From("sites").Where(Column("").Equals(1)).Build()
		if err == nil || err.Error() != "comparison column is required" {
			t.Fatalf("error = %v", err)
		}
	})
}
