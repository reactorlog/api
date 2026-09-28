package query

import (
	"errors"
	"fmt"
	"strings"
)

type InsertQuery struct {
	table   string
	columns []string
	values  []any
}

func Insert(table string) InsertQuery {
	return InsertQuery{
		table: table,
	}
}

func (q InsertQuery) Columns(columns ...string) InsertQuery {
	q.columns = columns
	return q
}

func (q InsertQuery) Values(values ...any) InsertQuery {
	q.values = values
	return q
}

func (q InsertQuery) validate() error {
	switch {
	case q.table == "":
		return errors.New("insert table is required")
	case len(q.columns) == 0:
		return errors.New("insert columns are required")
	case len(q.values) != len(q.columns):
		return errors.New("insert values must match columns")
	}
	return nil
}

func (q InsertQuery) Build() (string, []any, error) {
	if err := q.validate(); err != nil {
		return "", nil, err
	}

	b := &builder{}

	placeholders := make([]string, len(q.values))
	for i, value := range q.values {
		placeholders[i] = b.bind(value)
	}

	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		q.table,
		strings.Join(q.columns, ", "),
		strings.Join(placeholders, ", "),
	)
	return sql, b.args, nil
}
