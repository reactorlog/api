package query

import (
	"errors"
	"fmt"
	"strings"
)

type SelectQuery struct {
	columns []string
	table   string
	where   Condition
}

func Select(columns ...string) SelectQuery {
	return SelectQuery{
		columns: columns,
	}
}

func (q SelectQuery) From(table string) SelectQuery {
	q.table = table
	return q
}

func (q SelectQuery) Where(condition Condition) SelectQuery {
	q.where = condition
	return q
}

func (q SelectQuery) validate() error {
	switch {
	case len(q.columns) == 0:
		return errors.New("select columns are required")
	case q.table == "":
		return errors.New("select table is required")
	default:
		return nil
	}
}

func (q SelectQuery) Build() (string, []any, error) {
	if err := q.validate(); err != nil {
		return "", nil, err
	}

	sql := fmt.Sprintf("SELECT %s FROM %s", strings.Join(q.columns, ", "), q.table)
	if q.where == nil {
		return sql, nil, nil
	}

	where, args, err := q.where.build(1)
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("%s WHERE %s", sql, where), args, nil
}
