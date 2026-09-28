package schema

type Table struct {
	Name        string
	Columns     []Column
	IfNotExists bool
}
