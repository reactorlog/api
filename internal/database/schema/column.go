package schema

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
