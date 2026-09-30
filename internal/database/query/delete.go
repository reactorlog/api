package query

import "github.com/reactorlog/api/internal/database/expr"

type DeleteQuery struct {
	table string
	where expr.Condition
}

func Delete(table string) DeleteQuery {
	return DeleteQuery{
		table: table,
	}
}
