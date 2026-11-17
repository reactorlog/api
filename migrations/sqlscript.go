package migrations

import (
	"context"
	"database/sql"
	"strings"
)

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func splitSQL(script string) []string {
	var statements []string
	var b strings.Builder
	inSingle := false
	for i := 0; i < len(script); i++ {
		i, inSingle = consumeSQL(script, i, &b, inSingle, &statements)
	}
	appendStatement(&statements, &b)
	return statements
}

func consumeSQL(script string, i int, b *strings.Builder, inSingle bool, statements *[]string) (int, bool) {
	c := script[i]
	if c == '\'' {
		return consumeQuote(script, i, b, inSingle)
	}
	if c == ';' && !inSingle {
		appendStatement(statements, b)
		b.Reset()
		return i, inSingle
	}
	b.WriteByte(c)
	return i, inSingle
}

func consumeQuote(script string, i int, b *strings.Builder, inSingle bool) (int, bool) {
	if escapedQuote(script, i, inSingle) {
		b.WriteString("''")
		return i + 1, inSingle
	}
	b.WriteByte('\'')
	return i, !inSingle
}

func escapedQuote(script string, i int, inSingle bool) bool {
	return inSingle && i+1 < len(script) && script[i+1] == '\''
}

func appendStatement(statements *[]string, b *strings.Builder) {
	if statement := strings.TrimSpace(b.String()); statement != "" {
		*statements = append(*statements, statement)
	}
}

func execSQL(ctx context.Context, db execer, script string) error {
	for _, statement := range splitSQL(script) {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
