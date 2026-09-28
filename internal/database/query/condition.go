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
	And LogicalOperator = "AND"
	Or  LogicalOperator = "OR"
)

type Condition struct {
	comparison *Comparison
	logical    *Logical
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

type ConditionBuilder struct {
	column string
}

func Column(name string) ConditionBuilder {
	return ConditionBuilder{
		column: name,
	}
}

func (b ConditionBuilder) Equals(value interface{}) Condition {
	return b.comparison(Equal, value)
}

func (b ConditionBuilder) NotEquals(value interface{}) Condition {
	return b.comparison(NotEqual, value)
}

func (b ConditionBuilder) GreaterThan(value interface{}) Condition {
	return b.comparison(GreaterThan, value)
}

func (b ConditionBuilder) GreaterThanOrEqual(value interface{}) Condition {
	return b.comparison(GreaterThanOrEqual, value)
}

func (b ConditionBuilder) LessThan(value interface{}) Condition {
	return b.comparison(LessThan, value)
}

func (b ConditionBuilder) LessThanOrEqual(value interface{}) Condition {
	return b.comparison(LessThanOrEqual, value)
}

func (b ConditionBuilder) comparison(operator Operator, value interface{}) Condition {
	return Condition{
		comparison: &Comparison{
			column:   b.column,
			operator: operator,
			value:    value,
		},
	}
}

func (c Condition) And(right Condition) Condition {
	return c.combine(And, right)
}

func (c Condition) Or(right Condition) Condition {
	return c.combine(Or, right)
}

func (c Condition) combine(operator LogicalOperator, right Condition) Condition {
	return Condition{
		logical: &Logical{
			left:     c,
			operator: operator,
			right:    right,
		},
	}
}

func buildComparison(comparison Comparison, index int) (string, []any) {
	sql := fmt.Sprintf(
		"%s %s $%d",
		comparison.column,
		comparison.operator,
		index,
	)
	return sql, []any{comparison.value}
}

func buildLogical(logical Logical, index int) (string, []any, error) {
	leftSQL, leftArgs, err := buildCondition(logical.left, index)
	if err != nil {
		return "", nil, err
	}

	rightSQL, rightArgs, err := buildCondition(logical.right, index+len(leftArgs))
	if err != nil {
		return "", nil, err
	}

	sql := fmt.Sprintf(
		"(%s %s %s)",
		leftSQL,
		logical.operator,
		rightSQL,
	)
	return sql, append(leftArgs, rightArgs...), nil
}

func buildCondition(condition Condition, index int) (string, []any, error) {
	hasComparison := condition.comparison != nil
	hasLogical := condition.logical != nil

	if hasComparison && hasLogical {
		return "", nil, errors.New("condition type cannot be both comparison and logical")
	}
	if hasComparison {
		sql, args := buildComparison(*condition.comparison, index)
		return sql, args, nil
	}
	if hasLogical {
		return buildLogical(*condition.logical, index)
	}
	return "", nil, errors.New("condition is empty")
}
