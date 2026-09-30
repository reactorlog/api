package query

import (
	"fmt"

	"github.com/reactorlog/api/internal/database/expr"
)

type builder struct {
	args []any
}

func (b *builder) bind(value any) string {
	b.args = append(b.args, value)
	return fmt.Sprintf("$%d", len(b.args))
}

func (b *builder) formatValue(value any) (string, error) {
	return b.bind(value), nil
}

func (b *builder) buildCondition(condition Condition) (string, error) {
	return expr.Build(condition, b.formatValue)
}
