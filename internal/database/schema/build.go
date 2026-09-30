package schema

import (
	"errors"
	"fmt"
	"strings"
)

func validateColumn(c Column) error {
	if c.name == "" {
		return errors.New("column name is required")
	}
	if c.columnType == "" {
		return errors.New("column type is required")
	}
	return validateReference(c.reference)
}

func validateReference(reference *Reference) error {
	if reference == nil {
		return nil
	}
	if reference.table == "" {
		return errors.New("reference table is required")
	}
	if reference.column == "" {
		return errors.New("reference column is required")
	}
	return nil
}

func columnKeyConstraint(c Column) string {
	switch {
	case c.primaryKey:
		return "PRIMARY KEY"
	case c.notNull:
		return "NOT NULL"
	default:
		return ""
	}
}

func columnModifiers(c Column) []string {
	var parts []string
	if key := columnKeyConstraint(c); key != "" {
		parts = append(parts, key)
	}
	if c.unique {
		parts = append(parts, "UNIQUE")
	}
	if c.defaultType != "" {
		parts = append(parts, "DEFAULT "+string(c.defaultType))
	}
	if c.reference != nil {
		parts = append(parts, fmt.Sprintf("REFERENCES %s(%s)", c.reference.table, c.reference.column))
	}
	return parts
}

func buildColumn(c Column) (string, error) {
	if err := validateColumn(c); err != nil {
		return "", err
	}

	parts := []string{c.name, string(c.columnType)}
	parts = append(parts, columnModifiers(c)...)
	return strings.Join(parts, " "), nil
}

func validateTable(t Table) error {
	if t.Name == "" {
		return errors.New("table name is required")
	}
	if len(t.Columns) == 0 {
		return errors.New("table columns are required")
	}
	return nil
}

func buildColumnDefinitions(columns []Column) ([]string, error) {
	definitions := make([]string, 0, len(columns))
	for _, c := range columns {
		definition, err := buildColumn(c)
		if err != nil {
			return nil, fmt.Errorf("build column %q: %w", c.name, err)
		}
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

func createTablePrefix(t Table) string {
	prefix := "CREATE TABLE"
	if t.IfNotExists {
		prefix += " IF NOT EXISTS"
	}
	return prefix
}

func Build(t Table) (string, error) {
	if err := validateTable(t); err != nil {
		return "", err
	}

	columns, err := buildColumnDefinitions(t.Columns)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s %s (%s)", createTablePrefix(t), t.Name, strings.Join(columns, ", ")), nil
}

func Drop(t Table) (string, error) {
	if t.Name == "" {
		return "", errors.New("table name is required")
	}
	return fmt.Sprintf("DROP TABLE %s", t.Name), nil
}
