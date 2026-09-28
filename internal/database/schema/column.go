package schema

type columnType string

const (
	uuidType      columnType = "UUID"
	intType       columnType = "INT"
	bigIntType    columnType = "BIGINT"
	textType      columnType = "TEXT"
	timestampType columnType = "TIMESTAMP"
)

type defaultType string

const (
	uuidv7Default    defaultType = "uuidv7()"
	currentTimestamp defaultType = "CURRENT_TIMESTAMP"
)

type Column struct {
	name        string
	columnType  columnType
	notNull     bool
	primaryKey  bool
	unique      bool
	defaultType defaultType
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

func Text(name string) Column {
	return newColumn(name, textType)
}

func Timestamp(name string) Column {
	return newColumn(name, timestampType)
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

func (c Column) DefaultCurrentTimestamp() Column {
	c.defaultType = currentTimestamp
	return c
}

func (c Column) DefaultUUIDV7() Column {
	c.defaultType = uuidv7Default
	return c
}
