package eia

import "strings"

var softSuffixes = []string{
	"generating station",
	"generation station",
	"nuclear power plant",
	"nuclear station",
	"nuclear plant",
	"nuclear facility",
	"power station",
	"clean energy center",
	"nuclear",
	"project",
}

var softPrefixes = []string{
	"pseg ",
	"talenenergy ",
	"donald c ",
	"joseph m ",
	"james a ",
	"edwin i ",
	"r e ",
	"h b ",
	"v c ",
}

// SoftTokens is the deterministic secondary key for plant and site names.
func SoftTokens(name string) []string {
	s := softBase(stripParens(name))
	s = strings.ReplaceAll(s, "saint ", "st ")
	s = strings.Join(strings.Fields(s), " ")
	return strings.Fields(stripAffixes(s))
}

func SoftCompact(name string) string {
	return strings.Join(SoftTokens(name), "")
}

func softMatch(siteName, plantName string) bool {
	siteToks := SoftTokens(siteName)
	plantToks := SoftTokens(plantName)
	if len(siteToks) == 0 || len(plantToks) == 0 {
		return false
	}
	if SoftCompact(siteName) == SoftCompact(plantName) {
		return true
	}
	return tokenSubset(siteToks, plantToks) || tokenSubset(plantToks, siteToks)
}

func stripAffixes(s string) string {
	for {
		next := stripPrefixes(stripSuffixes(s))
		if next == s {
			return s
		}
		s = next
	}
}

func stripPrefixes(s string) string {
	for _, prefix := range softPrefixes {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimSpace(s[len(prefix):])
		}
	}
	return s
}

func stripSuffixes(s string) string {
	for _, suffix := range softSuffixes {
		if next, ok := stripOneSuffix(s, suffix); ok {
			return next
		}
	}
	return s
}

func stripOneSuffix(s, suffix string) (string, bool) {
	if s == suffix {
		return "", true
	}
	if strings.HasSuffix(s, " "+suffix) {
		return strings.TrimSpace(s[:len(s)-len(suffix)]), true
	}
	if strings.HasSuffix(s, suffix) && len(s) > len(suffix) {
		return strings.TrimSpace(s[:len(s)-len(suffix)]), true
	}
	return s, false
}

func stripParens(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		depth = writeOutsideParens(&b, r, depth)
	}
	return b.String()
}

func writeOutsideParens(b *strings.Builder, r rune, depth int) int {
	switch r {
	case '(':
		return depth + 1
	case ')':
		if depth > 0 {
			return depth - 1
		}
		return depth
	default:
		if depth == 0 {
			b.WriteRune(r)
		}
		return depth
	}
}

func tokenSubset(a, b []string) bool {
	set := make(map[string]bool, len(b))
	for _, t := range b {
		set[t] = true
	}
	for _, t := range a {
		if !set[t] {
			return false
		}
	}
	return true
}
