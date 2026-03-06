package connection

import "strings"

// QuoteLiteral wraps s as a SQL string literal, doubling embedded single quotes.
// Assumes standard_conforming_strings is on (PostgreSQL default since 9.1).
func QuoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// QuoteIdentifier wraps s as a SQL identifier, doubling embedded double quotes.
func QuoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
