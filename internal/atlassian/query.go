package atlassian

import "strings"

// quoteLiteral renders s as a quoted string literal for JQL and CQL, which
// share the same escaping rules: a backslash escapes a quote or another
// backslash.
//
// Without this, a value carrying a quote — a module name, an issue key typed by
// hand, a space key — either breaks the query with a parse error or, worse,
// closes the literal early and changes what the query actually asks for. Every
// interpolated value must go through here instead of being placed inside
// hand-written quotes in a format string.
//
// Control characters are folded to spaces: neither language accepts a raw
// newline inside a literal.
func quoteLiteral(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n', '\r', '\t':
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
