package postgres

import (
	"slices"
	"strings"
	"testing"
)

func TestParseWhereRebasesSnapshotColumnReferences(t *testing.T) {
	parser := newSnapshotWhereParser()
	parser.columnQualifier = "snapshot_row"
	parser.tableQualifiers = map[string]bool{"users": true, `Odd"Table`: true}
	for _, where := range []string{`id > 1`, `users.id > 1`, `"Odd""Table".id > 1`} {
		got, err := parser.Parse(where)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", where, err)
		}
		if want := `("snapshot_row"."id" > 1)`; got != want {
			t.Errorf("Parse(%q) = %q, want %q", where, got, want)
		}
	}
	got, err := parser.Parse(`name = 'users.id'`)
	if want := `("snapshot_row"."name" = E'users.id')`; err != nil || got != want {
		t.Errorf("Parse() = %q, %v, want %q", got, err, want)
	}
	got, err = parser.Parse(`unknown.id > 1`)
	if want := `("unknown"."id" > 1)`; err != nil || got != want {
		t.Errorf("Parse() = %q, %v, want unknown qualifier preserved as %q", got, err, want)
	}
}

func TestParseWhereRendersAcceptedExpressions(t *testing.T) {
	tests := []struct {
		name  string
		where string
		want  string
	}{
		{name: "empty", where: "", want: ""},
		{name: "blank", where: " \t\r\n", want: ""},
		{name: "comparison", where: "id = 1", want: `("id" = 1)`},
		{name: "every comparison operator", where: "a <> 1 or a != 1 or a < 1 or a > 1 or a <= 1 or a >= 1",
			want: `(((((("a" <> 1) OR ("a" != 1)) OR ("a" < 1)) OR ("a" > 1)) OR ("a" <= 1)) OR ("a" >= 1))`},
		{name: "bare identifier folds to lower case", where: "UserId = 1", want: `("userid" = 1)`},
		{name: "quoted identifier keeps case and escapes", where: `"User""Id" = 1`, want: `("User""Id" = 1)`},
		{name: "qualified column", where: `t.id = "T"."Id"`, want: `("t"."id" = "T"."Id")`},
		{name: "keyword function without parentheses is a column", where: "current_user = 'x'", want: `("current_user" = E'x')`},
		{name: "string escapes", where: "name = 'O''Brien'", want: `("name" = E'O''Brien')`},
		{name: "empty string", where: `name = ''`, want: `("name" = E'')`},
		{name: "literal backslash escapes", where: `name = '\n\t\x41\101'`, want: `("name" = E'\\n\\t\\x41\\101')`},
		{name: "trailing backslash", where: `name = 'path\'`, want: `("name" = E'path\\')`},
		{name: "backslash and apostrophe", where: `name = '\'''`, want: `("name" = E'\\''')`},
		{name: "injection shaped literal", where: `name = '\''; DROP TABLE users; --'`, want: `("name" = E'\\''; DROP TABLE users; --')`},
		{name: "typed literal backslash", where: `ts > timestamp '2024\01''02'`, want: `("ts" > TIMESTAMP E'2024\\01''02')`},
		{name: "numbers", where: "a = 1.5 or a = .5 or a = 2. or a = 1e10 or a = 1E+3 or a = 1.5e-2",
			want: `(((((("a" = 1.5) OR ("a" = .5)) OR ("a" = 2.)) OR ("a" = 1e10)) OR ("a" = 1E+3)) OR ("a" = 1.5e-2))`},
		{name: "keyword literals", where: "a IS NULL AND b = TRUE AND c = false", want: `((("a" IS NULL) AND ("b" = TRUE)) AND ("c" = FALSE))`},
		{name: "and binds tighter than or", where: "a = 1 or b = 2 and c = 3", want: `(("a" = 1) OR (("b" = 2) AND ("c" = 3)))`},
		{name: "grouping", where: "(a = 1) AND ((b = 2) OR (c = 3))", want: `((("a" = 1)) AND (((("b" = 2)) OR (("c" = 3)))))`},
		{name: "not", where: "NOT not a = 1", want: `(NOT (NOT ("a" = 1)))`},
		{name: "is tests", where: "a is not null and b is true and c is not false",
			want: `((("a" IS NOT NULL) AND ("b" IS TRUE)) AND ("c" IS NOT FALSE))`},
		{name: "is distinct from", where: "a is distinct from b or a is not distinct from 1",
			want: `(("a" IS DISTINCT FROM "b") OR ("a" IS NOT DISTINCT FROM 1))`},
		{name: "in", where: "a in (1, 2) and b not in ('x')", want: `(("a" IN (1, 2)) AND ("b" NOT IN (E'x')))`},
		{name: "between", where: "a between 1 and 2 and b not between 3 and 4",
			want: `(("a" BETWEEN 1 AND 2) AND ("b" NOT BETWEEN 3 AND 4))`},
		{name: "like", where: "a like 'x%' or a not like 'y' or a ilike 'z' or a not ilike 'w'",
			want: `(((("a" LIKE E'x%') OR ("a" NOT LIKE E'y')) OR ("a" ILIKE E'z')) OR ("a" NOT ILIKE E'w'))`},
		{name: "arithmetic precedence", where: "a + b * c - d / e % f = 0", want: `((("a" + ("b" * "c")) - (("d" / "e") % "f")) = 0)`},
		{name: "concatenation binds looser than arithmetic", where: "a || b + 1 = 'x'", want: `(("a" || ("b" + 1)) = E'x')`},
		{name: "unary signs", where: "a = - -1 and b = +2", want: `(("a" = (-((-(1))))) AND ("b" = (+(2))))`},
		{name: "cast binds tighter than unary minus", where: "-a::int = 1", want: `((-(("a"::int))) = 1)`},
		{name: "chained casts", where: "a::text::varchar(10) = 'x'", want: `((("a"::text)::varchar(10)) = E'x')`},
		{name: "multi-word types", where: "a::double precision = 1 and b::character varying = 'x' and c::character(2) = 'y'",
			want: `(((("a"::double precision) = 1) AND (("b"::character varying) = E'x')) AND (("c"::character(2)) = E'y'))`},
		{name: "types with time zone", where: "a::timestamp with time zone = b::time(3) without time zone",
			want: `(("a"::timestamp with time zone) = ("b"::time(3) without time zone))`},
		{name: "timestamp without zone suffix", where: "a::timestamp = b::time", want: `(("a"::timestamp) = ("b"::time))`},
		{name: "numeric precision", where: "a::numeric(10, 2) = 1", want: `(("a"::numeric(10,2)) = 1)`},
		{name: "cast call", where: "CAST(a AS bigint) = 1", want: `(CAST("a" AS bigint) = 1)`},
		{name: "extract", where: "extract(year from created_at) = 2024", want: `(EXTRACT(YEAR FROM "created_at") = 2024)`},
		{name: "typed literals", where: "ts > now() - interval '1 day' and d >= date '2024-01-01' and t < timestamptz '2024-01-01 00:00Z'",
			want: `((("ts" > (now() - INTERVAL E'1 day')) AND ("d" >= DATE E'2024-01-01')) AND ("t" < TIMESTAMPTZ E'2024-01-01 00:00Z'))`},
		{name: "current time keywords", where: "ts < current_timestamp and d <= CURRENT_DATE", want: `(("ts" < CURRENT_TIMESTAMP) AND ("d" <= CURRENT_DATE))`},
		{name: "functions", where: "lower(name) = 'x' and date_trunc('day', ts) = to_timestamp(0) and coalesce(a, b, 0) = 1",
			want: `(((lower("name") = E'x') AND (date_trunc(E'day', "ts") = to_timestamp(0))) AND (coalesce("a", "b", 0) = 1))`},
		{name: "function name is case-insensitive", where: "UPPER(a) = 'X'", want: `(upper("a") = E'X')`},
		{name: "type name as a column", where: "date = 1", want: `("date" = 1)`},
		{name: "keyword column", where: `"select" = 1`, want: `("select" = 1)`},
		{name: "bare value", where: "active", want: `"active"`},
		{name: "nesting at the limit", where: strings.Repeat("(", snapshotWhereMaxDepth-1) + "a" + strings.Repeat(")", snapshotWhereMaxDepth-1),
			want: strings.Repeat("(", snapshotWhereMaxDepth-1) + `"a"` + strings.Repeat(")", snapshotWhereMaxDepth-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newSnapshotWhereParser().Parse(tt.where)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.where, err)
			}

			if got != tt.want {
				t.Errorf("Parse(%q)\n got = %s\nwant = %s", tt.where, got, tt.want)
			}
		})
	}
}

func TestParseWhereRejectsInvalidExpressions(t *testing.T) {
	tests := []struct {
		name  string
		where string
		want  string
	}{
		{name: "statement separator", where: "a = 1; DROP TABLE users", want: `unexpected character ';'`},
		{name: "line comment", where: "a = 1 -- x", want: "comment at offset 6"},
		{name: "block comment", where: "a = 1 /* x */", want: "comment at offset 6"},
		{name: "dollar quoting", where: "a = $$x$$", want: `unexpected character '$'`},
		{name: "positional parameter", where: "a = $1", want: `unexpected character '$'`},
		{name: "regex operator", where: "a ~ 'x'", want: `unexpected character '~'`},
		{name: "array subscript", where: "a[1] = 1", want: `unexpected character '['`},
		{name: "non-ascii identifier", where: "имя = 1", want: "unexpected character"},
		{name: "unterminated string", where: "a = 'x", want: "unterminated quote at offset 4"},
		{name: "unterminated identifier", where: `"a = 1`, want: "unterminated quote at offset 0"},
		{name: "empty quoted identifier", where: `"" = 1`, want: "empty quoted identifier at offset 0"},
		{name: "malformed exponent", where: "a = 1e", want: "malformed number at offset 4"},
		{name: "escape string prefix", where: `a = E'x'`, want: `unexpected "x" at offset 5`},
		{name: "lowercase escape string prefix", where: `a = e'x'`, want: `unexpected "x" at offset 5`},
		{name: "typed escape string prefix", where: `a = interval E'7 days'`, want: `unexpected "e"`},
		{name: "subquery", where: "a in (select id from t)", want: `unexpected "id"`},
		{name: "exists subquery", where: "exists (select 1)", want: `function "exists" at offset 0 is not allowed`},
		{name: "disallowed function", where: "pg_sleep(10) is null", want: `function "pg_sleep" at offset 0 is not allowed`},
		{name: "schema-qualified function", where: "pg_catalog.lower(a) = 'x'", want: `unexpected "("`},
		{name: "quoted function name", where: `"lower"(a) = 'x'`, want: `unexpected "("`},
		{name: "star", where: "* = 1", want: `unexpected "*"`},
		{name: "trailing token", where: "a = 1 b", want: `unexpected "b" at offset 6`},
		{name: "chained comparison", where: "a = 1 = 2", want: `unexpected "=" at offset 6`},
		{name: "dangling operator", where: "a =", want: "unexpected end of expression at offset 3"},
		{name: "dangling or", where: "a = 1 or", want: "unexpected end of expression"},
		{name: "dangling and", where: "a = 1 and", want: "unexpected end of expression"},
		{name: "dangling not", where: "not", want: "unexpected end of expression"},
		{name: "dangling minus", where: "a = -", want: "unexpected end of expression"},
		{name: "dangling arithmetic", where: "a + ", want: "unexpected end of expression"},
		{name: "not without a test", where: "a not null", want: `unexpected "not"`},
		{name: "is without a test", where: "a is 1", want: `unexpected "1"`},
		{name: "is distinct without from", where: "a is distinct b", want: `unexpected "b"`},
		{name: "is distinct from without value", where: "a is distinct from", want: "unexpected end of expression"},
		{name: "in without list", where: "a in 1", want: `unexpected "1"`},
		{name: "empty in list", where: "a in ()", want: `unexpected ")"`},
		{name: "unclosed in list", where: "a in (1, 2", want: "unexpected end of expression"},
		{name: "in list with invalid item", where: "a in (1, )", want: `unexpected ")"`},
		{name: "between without and", where: "a between 1 or 2", want: `unexpected "or"`},
		{name: "between without low", where: "a between", want: "unexpected end of expression"},
		{name: "between without high", where: "a between 1 and", want: "unexpected end of expression"},
		{name: "like without pattern", where: "a like", want: "unexpected end of expression"},
		{name: "unclosed parenthesis", where: "(a = 1", want: "unexpected end of expression"},
		{name: "invalid parenthesized expression", where: "(a =)", want: `unexpected ")"`},
		{name: "stray closing parenthesis", where: ") = 1", want: `unexpected ")"`},
		{name: "disallowed cast type", where: "a::regclass = 1", want: "type at offset 3 is not allowed"},
		{name: "cast to non-word", where: "a::'x' = 1", want: "type at offset 3 is not allowed"},
		{name: "double without precision", where: "a::double = 1", want: `unexpected "="`},
		{name: "precision without number", where: "a::numeric(x) = 1", want: `unexpected "x"`},
		{name: "unclosed precision", where: "a::numeric(10 = 1", want: `unexpected "="`},
		{name: "with without time", where: "a::timestamp with zone = 1", want: `unexpected "zone"`},
		{name: "with time without zone", where: "a::time with time = 1", want: `unexpected "="`},
		{name: "cast call with invalid value", where: "cast(, int)", want: `unexpected ","`},
		{name: "cast call without as", where: "cast(a int)", want: `unexpected "int"`},
		{name: "cast call with invalid type", where: "cast(a as regclass)", want: "type at offset 10 is not allowed"},
		{name: "unclosed cast call", where: "cast(a as int", want: "unexpected end of expression"},
		{name: "extract with unknown field", where: "extract(foo from ts) = 1", want: `unexpected "foo"`},
		{name: "extract with string field", where: "extract('year' from ts) = 1", want: `unexpected "year"`},
		{name: "extract without from", where: "extract(year ts) = 1", want: `unexpected "ts"`},
		{name: "extract with invalid value", where: "extract(year from) = 1", want: `unexpected ")"`},
		{name: "unclosed extract", where: "extract(year from ts = 1", want: "unexpected end of expression"},
		{name: "function with invalid argument", where: "lower(;)", want: `unexpected character ';'`},
		{name: "function with dangling argument", where: "lower(a,", want: "unexpected end of expression"},
		{name: "qualified column without name", where: "t. = 1", want: `unexpected "="`},
		{name: "invalid cast after value", where: "a:: = 1", want: "type at offset 4 is not allowed"},
		{name: "nesting past the limit", where: strings.Repeat("(", snapshotWhereMaxDepth) + "a" + strings.Repeat(")", snapshotWhereMaxDepth),
			want: "expression is nested too deeply"},
		{name: "not chain past the limit", where: strings.Repeat("not ", snapshotWhereMaxDepth) + "a", want: "expression is nested too deeply"},
		{name: "sign chain past the limit", where: strings.Repeat("-", 2*snapshotWhereMaxDepth+1) + "1", want: "comment at offset 0"},
		{name: "spaced sign chain past the limit", where: strings.Repeat("- ", snapshotWhereMaxDepth) + "1", want: "expression is nested too deeply"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newSnapshotWhereParser().Parse(tt.where)
			if err == nil {
				t.Fatalf("Parse(%q) = %s, want error containing %q", tt.where, got, tt.want)
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Parse(%q) error = %q, want it to contain %q", tt.where, err, tt.want)
			}
		})
	}
}

func FuzzSnapshotWhereParser(f *testing.F) {
	for _, where := range []string{
		"",
		" \t\r\n",
		"id = 1",
		`"User""Id" = 1 and t.id = "T"."Id"`,
		"name = 'O''Brien' or a = 1.5e-2",
		`name = '\''; DROP TABLE users; --'`,
		`name = '\n\t\x41\101'`,
		`name = 'path\'`,
		"a is not distinct from b and c is not null",
		"a in (1, 2) and b not between 3 and 4 and c not ilike 'x%'",
		"a + b * c - d / e % f = 0 and a || b = 'x'",
		"a = - -1 and -a::int = 1",
		"a::timestamp with time zone = b::time(3) without time zone and c::numeric(10, 2) = 1",
		"CAST(a AS double precision) = 1 and extract(year from created_at) = 2024",
		"ts > now() - interval '1 day' and d >= date '2024-01-01'",
		"lower(name) = 'x' and coalesce(a, b, 0) = 1",
		"a = 1; DROP TABLE users",
		"a = 1 -- x",
		"a = 1 /* x */",
		"a = $$x$$",
		"pg_sleep(10) is null",
		"a in (select id from t)",
		`"" = 1`,
		"a = 'x",
		strings.Repeat("(", 70) + "a" + strings.Repeat(")", 70),
		strings.Repeat("not ", 70) + "a",
		strings.Repeat("-", 70) + "1 = 1",
	} {
		f.Add(where)
	}

	f.Fuzz(func(t *testing.T, where string) {
		got, err := newSnapshotWhereParser().Parse(where)

		reused := newSnapshotWhereParser()
		_, _ = reused.Parse("((a = 1 and")
		again, againErr := reused.Parse(where)
		if again != got || (againErr == nil) != (err == nil) {
			t.Fatalf("Parse(%q) on a reused parser = %q, %v, want %q, %v", where, again, againErr, got, err)
		}

		if err != nil {
			return
		}

		if got == "" {
			if strings.Trim(where, " \t\r\n") != "" {
				t.Fatalf("Parse(%q) = \"\", want a rendered filter for a non-blank input", where)
			}

			return
		}

		tokens := lexSnapshotWhereOutput(t, got)
		inputTokens, err := newSnapshotWhereLexer(where).lex()
		if err != nil {
			t.Fatalf("accepted input no longer lexes: %v", err)
		}
		var inputStrings, outputStrings []string
		for _, token := range inputTokens {
			if token.kind == snapshotWhereTokenString {
				inputStrings = append(inputStrings, token.text)
			}
		}
		for _, token := range tokens {
			if token.kind == snapshotWhereTokenString {
				outputStrings = append(outputStrings, token.text)
			}
		}
		if !slices.Equal(inputStrings, outputStrings) {
			t.Fatalf("Parse(%q) = %q changed string values: %q != %q", where, got, inputStrings, outputStrings)
		}

		depth := 0
		for _, token := range tokens {
			switch {
			case token.kind != snapshotWhereTokenSymbol:
			case token.text == "(":
				depth++
			case token.text == ")":
				depth--
				if depth < 0 {
					t.Fatalf("Parse(%q) = %q, which closes a parenthesis it did not open", where, got)
				}
			}
		}
		if depth != 0 {
			t.Fatalf("Parse(%q) = %q, which leaves %d parentheses open", where, got, depth)
		}
	})
}

func FuzzSnapshotWhereQuoting(f *testing.F) {
	f.Add("id", "1")
	f.Add("User\"Id", "O'Brien")
	f.Add("select", "'; DROP TABLE users; --")
	f.Add("a -- b", "/* x */")
	f.Add("\x00\xff", "\n\t\r")
	f.Add("path\\", `\'; DROP TABLE users; --`)
	f.Add("a", `\n\t\x41\101`)

	f.Fuzz(func(t *testing.T, name, value string) {
		if name == "" {
			t.Skip("the lexer rejects an empty quoted identifier by design")
		}

		where := quoteIdentifier(name) + " = '" + strings.ReplaceAll(value, "'", "''") + "'"

		got, err := newSnapshotWhereParser().Parse(where)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v, want a quoted comparison to be accepted", where, err)
		}

		if want := "(" + quoteIdentifier(name) + " = " + quoteLiteral(value) + ")"; got != want {
			t.Fatalf("Parse(%q) = %q, want %q", where, got, want)
		}
		tokens := lexSnapshotWhereOutput(t, got)
		if len(tokens) != 6 || tokens[1].kind != snapshotWhereTokenQuoted || tokens[1].text != name || tokens[3].kind != snapshotWhereTokenString || tokens[3].text != value {
			t.Fatalf("rendered comparison %q changed its identifier, value, or token boundaries: %+v", got, tokens)
		}
	})
}

// lexSnapshotWhereOutput checks escape-literal boundaries before lexing decoded
// output through the input lexer; the parser deliberately does not accept E literals.
func lexSnapshotWhereOutput(t *testing.T, sql string) []snapshotWhereToken {
	t.Helper()
	var normalized strings.Builder
	for i := 0; i < len(sql); {
		switch {
		case sql[i] == '"':
			start := i
			i++
			closed := false
			for i < len(sql) {
				if sql[i] != '"' {
					i++
					continue
				}
				i++
				if i < len(sql) && sql[i] == '"' {
					i++
					continue
				}
				closed = true
				break
			}
			if !closed {
				t.Fatalf("unterminated identifier in %q", sql)
			}
			normalized.WriteString(sql[start:i])
		case sql[i] == 'E' && i+1 < len(sql) && sql[i+1] == '\'':
			i += 2
			var value strings.Builder
			closed := false
			for i < len(sql) {
				switch sql[i] {
				case '\\':
					if i+1 >= len(sql) || sql[i+1] != '\\' {
						t.Fatalf("undoubled backslash in %q at %d", sql, i)
					}
					value.WriteByte('\\')
					i += 2
				case '\'':
					i++
					if i < len(sql) && sql[i] == '\'' {
						value.WriteByte('\'')
						i++
						continue
					}
					closed = true
				default:
					value.WriteByte(sql[i])
					i++
				}
				if closed {
					break
				}
			}
			if !closed {
				t.Fatalf("unterminated escape literal in %q", sql)
			}
			normalized.WriteByte('\'')
			normalized.WriteString(strings.ReplaceAll(value.String(), "'", "''"))
			normalized.WriteByte('\'')
		case sql[i] == '\'':
			t.Fatalf("non-escape literal in rendered SQL %q at %d", sql, i)
		default:
			normalized.WriteByte(sql[i])
			i++
		}
	}
	tokens, err := newSnapshotWhereLexer(normalized.String()).lex()
	if err != nil {
		t.Fatalf("rendered SQL %q has invalid token boundaries: %v", sql, err)
	}
	return tokens
}
