package query

import "github.com/reactorlog/api/internal/database/expr"

type Assignment struct {
	column string
	value  any
}

type UpdateQuery struct {
	table string
	set   []Assignment
	where expr.Condition
}

func Update(table string) UpdateQuery {
	return UpdateQuery{
		table: table,
	}
}
