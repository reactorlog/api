package query

import (
	"reflect"
	"testing"

	"github.com/reactorlog/api/internal/database/expr"
)

func BenchmarkSelectBuild(b *testing.B) {
	cases := []struct {
		name      string
		condition expr.Condition
		wantSQL   string
		wantArgs  []any
	}{
		{
			name:      "docket",
			condition: expr.Column("docket_number").Equals("05000424"),
			wantSQL:   "SELECT id, name, reactor_type FROM reactors WHERE docket_number = $1",
			wantArgs:  []any{"05000424"},
		},
		{
			name: "combined_filters",
			condition: expr.And(
				expr.Column("reactor_type").Equals("PWR"),
				expr.Or(expr.Column("unit_number").Equals("1"), expr.Column("unit_number").Equals("2")),
			),
			wantSQL:  "SELECT id, name, reactor_type FROM reactors WHERE (reactor_type = $1 AND (unit_number = $2 OR unit_number = $3))",
			wantArgs: []any{"PWR", "1", "2"},
		},
	}

	for _, test := range cases {
		b.Run(test.name, func(b *testing.B) {
			query := Select("id", "name", "reactor_type").From("reactors").Where(test.condition)
			var sql string
			var args []any
			var err error
			for b.Loop() {
				sql, args, err = query.Build()
				if err != nil {
					b.Fatal(err)
				}
			}
			if sql != test.wantSQL || !reflect.DeepEqual(args, test.wantArgs) {
				b.Fatalf("Build() = %q, %#v; want %q, %#v", sql, args, test.wantSQL, test.wantArgs)
			}
		})
	}
}

func BenchmarkInsertBuild(b *testing.B) {
	wantArgs := []any{"019a0000-0000-7000-8000-000000000001", "Vogtle 1", "1", "05000424", "PWR", "Westinghouse 4-loop"}
	query := Insert("reactors").
		Columns("site_id", "name", "unit_number", "docket_number", "reactor_type", "design").
		Values(wantArgs...)
	const wantSQL = "INSERT INTO reactors (site_id, name, unit_number, docket_number, reactor_type, design) VALUES ($1, $2, $3, $4, $5, $6)"
	var sql string
	var args []any
	var err error
	for b.Loop() {
		sql, args, err = query.Build()
		if err != nil {
			b.Fatal(err)
		}
	}
	if sql != wantSQL || !reflect.DeepEqual(args, wantArgs) {
		b.Fatalf("Build() = %q, %#v; want %q, %#v", sql, args, wantSQL, wantArgs)
	}
}
