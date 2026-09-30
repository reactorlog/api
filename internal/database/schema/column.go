package schema

import "github.com/reactorlog/api/internal/database/expr"

type columnType string

const (
	uuidType       columnType = "UUID"
	intType        columnType = "INT"
	bigIntType     columnType = "BIGINT"
	floatType      columnType = "FLOAT"
	textType       columnType = "TEXT"
	boolType       columnType = "BOOLEAN"
	timestampType  columnType = "TIMESTAMP"
	timestampzType columnType = "TIMESTAMPTZ"
)

type defaultType string

const (
	uuidv7Default    defaultType = "uuidv7()"
	currentTimestamp defaultType = "CURRENT_TIMESTAMP"
)

type Reference struct {
	table  string
	column string
}

type Column struct {
	name        string
	columnType  columnType
	notNull     bool
	primaryKey  bool
	unique      bool
	defaultType defaultType
	reference   *Reference
	checks      []expr.Condition
}

func newColumn(name string, columnType columnType) Column {
	return Column{
		name:       name,
		columnType: columnType,
	}
}

func Int(name string) Column {
	return newColumn(name, intType)
}

func BigInt(name string) Column {
	return newColumn(name, bigIntType)
}

func Float(name string) Column {
	return newColumn(name, floatType)
}

func Text(name string) Column {
	return newColumn(name, textType)
}

func Bool(name string) Column {
	return newColumn(name, boolType)
}

func Timestamp(name string) Column {
	return newColumn(name, timestampType)
}

func Timestampz(name string) Column {
	return newColumn(name, timestampzType)
}

func UUID(name string) Column {
	return newColumn(name, uuidType)
}

func (c Column) NotNull() Column {
	c.notNull = true
	return c
}

func (c Column) PrimaryKey() Column {
	c.primaryKey = true
	return c
}

func (c Column) Unique() Column {
	c.unique = true
	return c
}

func (c Column) References(table, column string) Column {
	c.reference = &Reference{
		table:  table,
		column: column,
	}
	return c
}

func (c Column) DefaultCurrentTimestamp() Column {
	c.defaultType = currentTimestamp
	return c
}

func (c Column) DefaultUUIDV7() Column {
	c.defaultType = uuidv7Default
	return c
}

// Check adds a condition that PostgreSQL enforces when rows are written.
func (c Column) Check(condition expr.Condition) Column {
	checks := make([]expr.Condition, len(c.checks)+1)
	copy(checks, c.checks)
	checks[len(c.checks)] = condition
	c.checks = checks
	return c
}

// Between adds inclusive bounds. Use NotNull separately to require a value.
func (c Column) Between(min, max float64) Column {
	column := expr.Column(c.name)
	return c.Check(expr.And(
		column.GreaterThanOrEqual(min),
		column.LessThanOrEqual(max),
	))
}
