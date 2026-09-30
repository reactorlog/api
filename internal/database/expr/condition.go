// Package expr provides conditions shared by SQL queries and schema checks.
package expr

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

// ValueFormatter renders a value as a query placeholder or a SQL literal.
type ValueFormatter func(any) (string, error)

type Condition interface {
	build(ValueFormatter) (string, error)
}

type Comparison struct {
	column   string
	operator Operator
	value    any
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

func (c ColumnRef) Equals(value any) Condition {
	return Comparison{column: string(c), operator: Equal, value: value}
}

func (c ColumnRef) NotEquals(value any) Condition {
	return Comparison{column: string(c), operator: NotEqual, value: value}
}

func (c ColumnRef) GreaterThan(value any) Condition {
	return Comparison{column: string(c), operator: GreaterThan, value: value}
}

func (c ColumnRef) GreaterThanOrEqual(value any) Condition {
	return Comparison{column: string(c), operator: GreaterThanOrEqual, value: value}
}

func (c ColumnRef) LessThan(value any) Condition {
	return Comparison{column: string(c), operator: LessThan, value: value}
}

func (c ColumnRef) LessThanOrEqual(value any) Condition {
	return Comparison{column: string(c), operator: LessThanOrEqual, value: value}
}

func And(left, right Condition) Condition {
	return Logical{left: left, operator: AndOperator, right: right}
}

func Or(left, right Condition) Condition {
	return Logical{left: left, operator: OrOperator, right: right}
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

func validateComparisonOperator(operator Operator) error {
	switch operator {
	case Equal, NotEqual, GreaterThan, GreaterThanOrEqual, LessThan, LessThanOrEqual:
		return nil
	default:
		return errors.New("invalid comparison operator")
	}
}

func (c Comparison) build(formatValue ValueFormatter) (string, error) {
	if c.column == "" {
		return "", errors.New("comparison column is required")
	}
	if err := validateComparisonOperator(c.operator); err != nil {
		return "", err
	}
	value, err := formatValue(c.value)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %s %s", c.column, c.operator, value), nil
}

func validateLogicalOperator(operator LogicalOperator) error {
	switch operator {
	case AndOperator, OrOperator:
		return nil
	default:
		return errors.New("invalid logical operator")
	}
}

func (l Logical) build(formatValue ValueFormatter) (string, error) {
	if err := validateLogicalOperator(l.operator); err != nil {
		return "", err
	}
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
