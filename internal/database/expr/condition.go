// Package expr provides conditions shared by SQL queries and schema checks.
package expr

import (
	"errors"
	"fmt"
)

type operator string

const (
	equal              operator = "="
	notEqual           operator = "!="
	greaterThan        operator = ">"
	greaterThanOrEqual operator = ">="
	lessThan           operator = "<"
	lessThanOrEqual    operator = "<="
)

type logicalOperator string

const (
	andOperator logicalOperator = "AND"
	orOperator  logicalOperator = "OR"
)

// ValueFormatter renders a value as a query placeholder or a SQL literal.
type ValueFormatter func(any) (string, error)

type Condition interface {
	build(ValueFormatter) (string, error)
}

type comparison struct {
	column   string
	operator operator
	value    any
}

type logical struct {
	left     Condition
	operator logicalOperator
	right    Condition
}

type ColumnRef string

func Column(name string) ColumnRef {
	return ColumnRef(name)
}

func (c ColumnRef) Equals(value any) Condition {
	return comparison{column: string(c), operator: equal, value: value}
}

func (c ColumnRef) NotEquals(value any) Condition {
	return comparison{column: string(c), operator: notEqual, value: value}
}

func (c ColumnRef) GreaterThan(value any) Condition {
	return comparison{column: string(c), operator: greaterThan, value: value}
}

func (c ColumnRef) GreaterThanOrEqual(value any) Condition {
	return comparison{column: string(c), operator: greaterThanOrEqual, value: value}
}

func (c ColumnRef) LessThan(value any) Condition {
	return comparison{column: string(c), operator: lessThan, value: value}
}

func (c ColumnRef) LessThanOrEqual(value any) Condition {
	return comparison{column: string(c), operator: lessThanOrEqual, value: value}
}

func And(left, right Condition) Condition {
	return logical{left: left, operator: andOperator, right: right}
}

func Or(left, right Condition) Condition {
	return logical{left: left, operator: orOperator, right: right}
}

// Build renders a condition using the caller's value formatter.
func Build(condition Condition, formatValue ValueFormatter) (string, error) {
	if condition == nil {
		return "", errors.New("condition is required")
	}
	if formatValue == nil {
		return "", errors.New("value formatter is required")
	}
	return condition.build(formatValue)
}

func (c comparison) build(formatValue ValueFormatter) (string, error) {
	if c.column == "" {
		return "", errors.New("comparison column is required")
	}
	value, err := formatValue(c.value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %s %s", c.column, c.operator, value), nil
}

func (l logical) build(formatValue ValueFormatter) (string, error) {
	leftSQL, err := Build(l.left, formatValue)
	if err != nil {
		return "", err
	}
	rightSQL, err := Build(l.right, formatValue)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(%s %s %s)", leftSQL, l.operator, rightSQL), nil
}
