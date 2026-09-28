package query

import "testing"

func TestSelectValidate(t *testing.T) {
	t.Run("missing columns", func(t *testing.T) {
		err := Select().From("sites").validate()
		if err == nil || err.Error() != "select columns are required" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("missing table", func(t *testing.T) {
		err := Select("id").validate()
		if err == nil || err.Error() != "select table is required" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("ok", func(t *testing.T) {
		if err := Select("id").From("sites").validate(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSelectBuild(t *testing.T) {
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
		sql, args, err := Select("id").
			From("sites").
			Where(Column("name").Equals("Ada")).
			Build()
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

	t.Run("validate error", func(t *testing.T) {
		_, _, err := Select().From("sites").Build()
		if err == nil || err.Error() != "select columns are required" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("where error", func(t *testing.T) {
		_, _, err := Select("id").From("sites").Where(Condition{}).Build()
		if err == nil || err.Error() != "condition is empty" {
			t.Fatalf("error = %v", err)
		}
	})
}
