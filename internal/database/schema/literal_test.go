package schema

import (
	"math"
	"testing"
)

func TestFormatLiteral(t *testing.T) {
	type namedFloat float64
	type namedString string
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{"null", nil, "NULL"},
		{"true", true, "true"},
		{"false", false, "false"},
		{"integer", int(-5), "-5"},
		{"int8", int8(-5), "-5"},
		{"int16", int16(-5), "-5"},
		{"int32", int32(-5), "-5"},
		{"int64", int64(-9223372036854775808), "-9223372036854775808"},
		{"uint", uint(5), "5"},
		{"uint8", uint8(5), "5"},
		{"uint16", uint16(5), "5"},
		{"uint32", uint32(5), "5"},
		{"uint64", uint64(18446744073709551615), "18446744073709551615"},
		{"float32", float32(0.1), "0.1"},
		{"float64", float64(-90.25), "-90.25"},
		{"named float", namedFloat(180), "180"},
		{"empty string", "", "E''"},
		{"string", "Ada", "E'Ada'"},
		{"named string", namedString("Ada"), "E'Ada'"},
		{"quote", "O'Brien", "E'O''Brien'"},
		{"backslash", `a\b`, `E'a\\b'`},
		{"quote and backslash", `\'; DROP TABLE sites; --`, `E'\\''; DROP TABLE sites; --'`},
		{"newline", "a\nb", "E'a\nb'"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := formatLiteral(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("literal = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatLiteralRejectsUnsafeValues(t *testing.T) {
	cases := []struct {
		name  string
		value any
	}{
		{"NaN", math.NaN()},
		{"positive infinity", math.Inf(1)},
		{"negative infinity", math.Inf(-1)},
		{"float32 NaN", float32(math.NaN())},
		{"NUL", "a\x00b"},
		{"slice", []byte("Ada")},
		{"struct", struct{ Name string }{"Ada"}},
		{"pointer", new(int)},
		{"nil pointer", (*int)(nil)},
		{"complex", complex(1, 2)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := formatLiteral(tc.value)
			if err == nil {
				t.Fatalf("expected error, got literal %q", got)
			}
			if got != "" {
				t.Fatalf("error returned partial literal %q", got)
			}
		})
	}
}
