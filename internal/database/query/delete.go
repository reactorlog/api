package query

type DeleteQuery struct {
	table string
	where *Condition
}

func Delete(table string) DeleteQuery {
	return DeleteQuery{
		table: table,
	}
}