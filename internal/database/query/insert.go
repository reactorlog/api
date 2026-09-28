package query

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
