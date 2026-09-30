package query

import (
	"errors"
	"fmt"
	"strings"

	"github.com/reactorlog/api/internal/database/expr"
)

type SelectQuery struct {
	columns []string
	table   string
	where   expr.Condition
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

func (q SelectQuery) Where(condition expr.Condition) SelectQuery {
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

	b := &builder{}

	sql := fmt.Sprintf(
		"SELECT %s FROM %s",
		strings.Join(q.columns, ", "),
		q.table,
	)
	if q.where != nil {
		where, err := b.buildCondition(q.where)
		if err != nil {
			return "", nil, err
		}
		sql += " WHERE " + where
	}

	return sql, b.args, nil
}
