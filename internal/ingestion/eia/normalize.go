package eia

import (
	"strings"
	"unicode"
)

// NormalizeName builds the MatchKey for exact site-to-plant association.
func NormalizeName(s string) string {
	return collapseRunes(strings.ToLower(strings.TrimSpace(s)), false)
}

func softBase(name string) string {
	return collapseRunes(strings.ToLower(strings.TrimSpace(name)), true)
}

// collapseRunes keeps letters and digits. Spaces are collapsed.
// When otherAsSpace is true, punctuation becomes a space; otherwise it is dropped.
func collapseRunes(s string, otherAsSpace bool) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		prevSpace = appendCollapsed(&b, r, otherAsSpace, prevSpace)
	}
	return strings.TrimSpace(b.String())
}

func appendCollapsed(b *strings.Builder, r rune, otherAsSpace, prevSpace bool) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		b.WriteRune(r)
		return false
	}
	if unicode.IsSpace(r) || otherAsSpace {
		return appendSpace(b, prevSpace)
	}
	return prevSpace
}

func appendSpace(b *strings.Builder, prevSpace bool) bool {
	if !prevSpace && b.Len() > 0 {
		b.WriteByte(' ')
		return true
	}
	return prevSpace
}
