package schema

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

type columnCase struct {
	name          string
	column        Column
	sql           string
	dataType      string
	notNull       bool
	columnDefault string
	primaryKey    bool
	unique        bool
}

func columnCases() []columnCase {
	return []columnCase{
		{
			name:     "uuid",
			column:   UUID("id"),
			sql:      "id UUID",
			dataType: "uuid",
		},
		{
			name:          "uuid primary key default",
			column:        UUID("id").PrimaryKey().DefaultUUIDV7(),
			sql:           "id UUID PRIMARY KEY DEFAULT uuidv7()",
			dataType:      "uuid",
			notNull:       true,
			columnDefault: "uuidv7()",
			primaryKey:    true,
		},
		{
			name:     "uuid not null",
			column:   UUID("id").NotNull(),
			sql:      "id UUID NOT NULL",
			dataType: "uuid",
			notNull:  true,
		},
		{
			// PRIMARY KEY implies NOT NULL; builder must not also emit NOT NULL.
			name:          "primary key with not null",
			column:        UUID("id").PrimaryKey().NotNull().DefaultUUIDV7(),
			sql:           "id UUID PRIMARY KEY DEFAULT uuidv7()",
			dataType:      "uuid",
			notNull:       true,
			columnDefault: "uuidv7()",
			primaryKey:    true,
		},
		{
			name:          "uuid default without primary key",
			column:        UUID("id").DefaultUUIDV7(),
			sql:           "id UUID DEFAULT uuidv7()",
			dataType:      "uuid",
			columnDefault: "uuidv7()",
		},
		{
			name:     "text",
			column:   Text("note"),
			sql:      "note TEXT",
			dataType: "text",
		},
		{
			name:     "text not null",
			column:   Text("name").NotNull(),
			sql:      "name TEXT NOT NULL",
			dataType: "text",
			notNull:  true,
		},
		{
			name:     "text unique",
			column:   Text("email").Unique(),
			sql:      "email TEXT UNIQUE",
			dataType: "text",
			unique:   true,
		},
		{
			name:     "bigint",
			column:   BigInt("count"),
			sql:      "count BIGINT",
			dataType: "bigint",
		},
		{
			name:     "bigint not null",
			column:   BigInt("sequence").NotNull(),
			sql:      "sequence BIGINT NOT NULL",
			dataType: "bigint",
			notNull:  true,
		},
		{
			name:       "bigint primary key",
			column:     BigInt("id").PrimaryKey(),
			sql:        "id BIGINT PRIMARY KEY",
			dataType:   "bigint",
			notNull:    true,
			primaryKey: true,
		},
		{
			name:     "int not null unique",
			column:   Int("version").NotNull().Unique(),
			sql:      "version INT NOT NULL UNIQUE",
			dataType: "integer",
			notNull:  true,
			unique:   true,
		},
		{
			name:     "timestamp",
			column:   Timestamp("occurred_at"),
			sql:      "occurred_at TIMESTAMP",
			dataType: "timestamp without time zone",
		},
		{
			name:     "timestamp not null",
			column:   Timestamp("locked_at").NotNull(),
			sql:      "locked_at TIMESTAMP NOT NULL",
			dataType: "timestamp without time zone",
			notNull:  true,
		},
		{
			name:          "timestamp not null default",
			column:        Timestamp("created_at").NotNull().DefaultCurrentTimestamp(),
			sql:           "created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP",
			dataType:      "timestamp without time zone",
			notNull:       true,
			columnDefault: "CURRENT_TIMESTAMP",
		},
		{
			name:          "timestamp default",
			column:        Timestamp("closed_at").DefaultCurrentTimestamp(),
			sql:           "closed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
			dataType:      "timestamp without time zone",
			columnDefault: "CURRENT_TIMESTAMP",
		},
		{
			name:          "timestamp unique default",
			column:        Timestamp("closed_at").Unique().DefaultCurrentTimestamp(),
			sql:           "closed_at TIMESTAMP UNIQUE DEFAULT CURRENT_TIMESTAMP",
			dataType:      "timestamp without time zone",
			columnDefault: "CURRENT_TIMESTAMP",
			unique:        true,
		},
	}
}

func TestBuildColumn(t *testing.T) {
	for _, tc := range columnCases() {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildColumn(tc.column)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.sql {
				t.Fatalf("SQL = %q, want %q", got, tc.sql)
			}
		})
	}

	t.Run("missing name", func(t *testing.T) {
		_, err := buildColumn(Column{columnType: textType})
		if err == nil || err.Error() != "column name is required" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("missing type", func(t *testing.T) {
		_, err := buildColumn(Column{name: "id"})
		if err == nil || err.Error() != "column type is required" {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestBuild(t *testing.T) {
	got, err := Build(Table{
		Name: "schema_build",
		Columns: []Column{
			UUID("id").PrimaryKey().DefaultUUIDV7(),
			Text("name").NotNull(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := "CREATE TABLE schema_build (id UUID PRIMARY KEY DEFAULT uuidv7(), name TEXT NOT NULL)"
	if got != want {
		t.Fatalf("SQL = %q, want %q", got, want)
	}

	t.Run("if not exists", func(t *testing.T) {
		got, err := Build(Table{
			Name:        "schema_build",
			IfNotExists: true,
			Columns:     []Column{Text("name").NotNull()},
		})
		if err != nil {
			t.Fatal(err)
		}
		want := "CREATE TABLE IF NOT EXISTS schema_build (name TEXT NOT NULL)"
		if got != want {
			t.Fatalf("SQL = %q, want %q", got, want)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		_, err := Build(Table{Columns: []Column{Text("name").NotNull()}})
		if err == nil || err.Error() != "table name is required" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("missing columns", func(t *testing.T) {
		_, err := Build(Table{Name: "schema_build"})
		if err == nil || err.Error() != "table columns are required" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("invalid column", func(t *testing.T) {
		_, err := Build(Table{Name: "schema_build", Columns: []Column{{}}})
		if err == nil || err.Error() != `build column "": column name is required` {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("invalid column preserves name in error", func(t *testing.T) {
		_, err := Build(Table{Name: "schema_build", Columns: []Column{{name: "broken"}}})
		if err == nil || err.Error() != `build column "broken": column type is required` {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestDrop(t *testing.T) {
	got, err := Drop(Table{Name: "schema_build"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "DROP TABLE schema_build" {
		t.Fatalf("SQL = %q, want %q", got, "DROP TABLE schema_build")
	}

	t.Run("ignores columns", func(t *testing.T) {
		got, err := Drop(Table{
			Name:    "schema_build",
			Columns: []Column{Text("name").NotNull()},
		})
		if err != nil {
			t.Fatal(err)
		}
		if got != "DROP TABLE schema_build" {
			t.Fatalf("SQL = %q, want %q", got, "DROP TABLE schema_build")
		}
	})

	t.Run("missing name", func(t *testing.T) {
		_, err := Drop(Table{})
		if err == nil || err.Error() != "table name is required" {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestGeneratedSQLIsValid(t *testing.T) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	for i, tc := range columnCases() {
		t.Run(tc.name, func(t *testing.T) {
			definition, err := buildColumn(tc.column)
			if err != nil {
				t.Fatal(err)
			}
			table := fmt.Sprintf("schema_build_col_%d", i)
			statement := fmt.Sprintf("CREATE TABLE %s (%s)", table, definition)
			if _, err := tx.Exec(ctx, statement); err != nil {
				t.Fatalf("exec %s: %v", statement, err)
			}
			assertColumn(t, tx, table, tc)
		})
	}

	t.Run("create table preserves column order", func(t *testing.T) {
		// One representative column per type; per-column metadata is covered above.
		columns := []Column{
			UUID("id").PrimaryKey().DefaultUUIDV7(),
			Text("name").NotNull(),
			BigInt("sequence").NotNull(),
			Timestamp("created_at").NotNull().DefaultCurrentTimestamp(),
		}

		statement, err := Build(Table{Name: "schema_build", Columns: columns})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, statement); err != nil {
			t.Fatalf("exec %s: %v", statement, err)
		}

		rows, err := tx.Query(ctx, `
			SELECT column_name
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'schema_build'
			ORDER BY ordinal_position
		`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()

		var names []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatal(err)
			}
			names = append(names, name)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}

		want := []string{"id", "name", "sequence", "created_at"}
		if len(names) != len(want) {
			t.Fatalf("columns = %v, want %v", names, want)
		}
		for i := range want {
			if names[i] != want[i] {
				t.Fatalf("column %d = %s, want %s", i, names[i], want[i])
			}
		}
	})

	t.Run("drop table", func(t *testing.T) {
		table := Table{
			Name: "schema_drop",
			Columns: []Column{
				UUID("id").PrimaryKey().DefaultUUIDV7(),
				Text("name").NotNull(),
			},
		}

		create, err := Build(table)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, create); err != nil {
			t.Fatalf("create: %v", err)
		}

		drop, err := Drop(table)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, drop); err != nil {
			t.Fatalf("drop: %v", err)
		}

		var exists bool
		err = tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = 'schema_drop'
			)
		`).Scan(&exists)
		if err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Fatal("table still exists after drop")
		}
	})
}

func assertColumn(t *testing.T, tx pgx.Tx, table string, tc columnCase) {
	t.Helper()
	ctx := context.Background()

	var dataType, isNullable string
	var columnDefault *string
	err := tx.QueryRow(ctx, `
		SELECT data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = $1
		  AND column_name = $2
	`, table, tc.column.name).Scan(&dataType, &isNullable, &columnDefault)
	if err != nil {
		t.Fatalf("read %s.%s: %v", table, tc.column.name, err)
	}

	notNull := isNullable == "NO"
	if dataType != tc.dataType || notNull != tc.notNull {
		t.Fatalf("%s.%s type=%s not_null=%v, want type=%s not_null=%v", table, tc.column.name, dataType, notNull, tc.dataType, tc.notNull)
	}

	gotDefault := ""
	if columnDefault != nil {
		gotDefault = *columnDefault
	}
	if gotDefault != tc.columnDefault {
		t.Fatalf("%s.%s default=%q, want %q", table, tc.column.name, gotDefault, tc.columnDefault)
	}

	var primaryKey string
	err = tx.QueryRow(ctx, `
		SELECT kcu.column_name
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema = kcu.table_schema
		 AND tc.table_name = kcu.table_name
		WHERE tc.table_schema = 'public'
		  AND tc.table_name = $1
		  AND tc.constraint_type = 'PRIMARY KEY'
		  AND kcu.column_name = $2
	`, table, tc.column.name).Scan(&primaryKey)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if tc.primaryKey {
			t.Fatalf("%s.%s is not a primary key", table, tc.column.name)
		}
	case err != nil:
		t.Fatalf("primary key %s.%s: %v", table, tc.column.name, err)
	default:
		if !tc.primaryKey {
			t.Fatalf("%s.%s is a primary key", table, tc.column.name)
		}
	}

	var unique string
	err = tx.QueryRow(ctx, `
		SELECT kcu.column_name
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema = kcu.table_schema
		 AND tc.table_name = kcu.table_name
		WHERE tc.table_schema = 'public'
		  AND tc.table_name = $1
		  AND tc.constraint_type = 'UNIQUE'
		  AND kcu.column_name = $2
	`, table, tc.column.name).Scan(&unique)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if tc.unique {
			t.Fatalf("%s.%s is not unique", table, tc.column.name)
		}
	case err != nil:
		t.Fatalf("unique %s.%s: %v", table, tc.column.name, err)
	default:
		if !tc.unique {
			t.Fatalf("%s.%s is unique", table, tc.column.name)
		}
	}
}

func databaseURL(t *testing.T) string {
	t.Helper()
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		file, err := os.Open(filepath.Join(dir, ".env"))
		if err == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				line = strings.TrimPrefix(line, "export ")
				key, value, ok := strings.Cut(line, "=")
				if ok && key == "DATABASE_URL" {
					file.Close()
					return strings.Trim(value, `"'`)
				}
			}
			file.Close()
			if err := scanner.Err(); err != nil {
				t.Fatal(err)
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	t.Fatal("DATABASE_URL is required")
	return ""
}
