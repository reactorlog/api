package query

type Assignment struct {
	column string
	value  any
}

type UpdateQuery struct {
	table string
	set   []Assignment
	where Condition
}

func Update(table string) UpdateQuery {
	return UpdateQuery{
		table: table,
	}
}
