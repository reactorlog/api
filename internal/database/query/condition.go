package query

import (
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

type Condition interface {
	build(index int) (string, []any, error)
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

func (c Comparison) build(index int) (string, []any, error) {
	sql := fmt.Sprintf(
		"%s %s $%d",
		c.column,
		c.operator,
		index,
	)
	return sql, []any{c.value}, nil
}

func (l Logical) build(index int) (string, []any, error) {
	leftSQL, leftArgs, err := l.left.build(index)
	if err != nil {
		return "", nil, err
	}
	rightSQL, rightArgs, err := l.right.build(index + len(leftArgs))
	if err != nil {
		return "", nil, err
	}
	sql := fmt.Sprintf(
		"(%s %s %s)",
		leftSQL,
		l.operator,
		rightSQL,
	)
	return sql, append(leftArgs, rightArgs...), nil
}
