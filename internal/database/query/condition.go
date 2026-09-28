package query

import (
	"errors"
	"fmt"
)

type Operator string

const (
	Equal              Operator = "="
	NotEqual           Operator = "!="
	GreaterThan        Operator = ">"
	GreaterThanOrEqual Operator = ">="
	LessThan           Operator = "<"
	LessThanOrEqual    Operator = "<="
)

type LogicalOperator string

const (
	AndOperator LogicalOperator = "AND"
	OrOperator  LogicalOperator = "OR"
)

type builder struct {
	args []any
}

func (b *builder) bind(value any) string {
	b.args = append(b.args, value)
	return fmt.Sprintf("$%d", len(b.args))
}

type Condition interface {
	build(*builder) (string, error)
}

type Comparison struct {
	column   string
	operator Operator
	value    interface{}
}

type Logical struct {
	left     Condition
	operator LogicalOperator
	right    Condition
}

type ColumnRef string

func Column(name string) ColumnRef {
	return ColumnRef(name)
}

func (c ColumnRef) Equals(value interface{}) Condition {
	return Comparison{
		column:   string(c),
		operator: Equal,
		value:    value,
	}
}

func (c ColumnRef) NotEquals(value interface{}) Condition {
	return Comparison{
		column:   string(c),
		operator: NotEqual,
		value:    value,
	}
}

func (c ColumnRef) GreaterThan(value interface{}) Condition {
	return Comparison{
		column:   string(c),
		operator: GreaterThan,
		value:    value,
	}
}

func (c ColumnRef) GreaterThanOrEqual(value interface{}) Condition {
	return Comparison{
		column:   string(c),
		operator: GreaterThanOrEqual,
		value:    value,
	}
}

func (c ColumnRef) LessThan(value interface{}) Condition {
	return Comparison{
		column:   string(c),
		operator: LessThan,
		value:    value,
	}
}

func (c ColumnRef) LessThanOrEqual(value interface{}) Condition {
	return Comparison{
		column:   string(c),
		operator: LessThanOrEqual,
		value:    value,
	}
}

func And(left, right Condition) Condition {
	return Logical{
		left:     left,
		operator: AndOperator,
		right:    right,
	}
}

func Or(left, right Condition) Condition {
	return Logical{
		left:     left,
		operator: OrOperator,
		right:    right,
	}
}

func (c Comparison) build(b *builder) (string, error) {
	if c.column == "" {
		return "", errors.New("comparison column is required")
	}

	placeholder := b.bind(c.value)

	sql := fmt.Sprintf(
		"%s %s %s",
		c.column,
		c.operator,
		placeholder,
	)
	return sql, nil
}

func (l Logical) build(b *builder) (string, error) {
	leftSQL, err := l.left.build(b)
	if err != nil {
		return "", err
	}
	rightSQL, err := l.right.build(b)
	if err != nil {
		return "", err
	}
	sql := fmt.Sprintf(
		"(%s %s %s)",
		leftSQL,
		l.operator,
		rightSQL,
	)
	return sql, nil
}
