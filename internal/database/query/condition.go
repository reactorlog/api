package query

import "github.com/reactorlog/api/internal/database/expr"

// Conditions are shared with schema checks. These aliases preserve the query API.
type Condition = expr.Condition
type Comparison = expr.Comparison
type Logical = expr.Logical
type ColumnRef = expr.ColumnRef
type Operator = expr.Operator
type LogicalOperator = expr.LogicalOperator

const (
	Equal              = expr.Equal
	NotEqual           = expr.NotEqual
	GreaterThan        = expr.GreaterThan
	GreaterThanOrEqual = expr.GreaterThanOrEqual
	LessThan           = expr.LessThan
	LessThanOrEqual    = expr.LessThanOrEqual
	AndOperator        = expr.AndOperator
	OrOperator         = expr.OrOperator
)

func Column(name string) ColumnRef {
	return expr.Column(name)
}

func And(left, right Condition) Condition {
	return expr.And(left, right)
}

func Or(left, right Condition) Condition {
	return expr.Or(left, right)
}
