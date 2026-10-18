package postgres

import (
	"strings"
	"testing"
)

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
		{name: "keyword function without parentheses is a column", where: "current_user = 'x'", want: `("current_user" = 'x')`},
		{name: "string escapes", where: "name = 'O''Brien'", want: `("name" = 'O''Brien')`},
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
		{name: "in", where: "a in (1, 2) and b not in ('x')", want: `(("a" IN (1, 2)) AND ("b" NOT IN ('x')))`},
		{name: "between", where: "a between 1 and 2 and b not between 3 and 4",
			want: `(("a" BETWEEN 1 AND 2) AND ("b" NOT BETWEEN 3 AND 4))`},
		{name: "like", where: "a like 'x%' or a not like 'y' or a ilike 'z' or a not ilike 'w'",
			want: `(((("a" LIKE 'x%') OR ("a" NOT LIKE 'y')) OR ("a" ILIKE 'z')) OR ("a" NOT ILIKE 'w'))`},
		{name: "arithmetic precedence", where: "a + b * c - d / e % f = 0", want: `((("a" + ("b" * "c")) - (("d" / "e") % "f")) = 0)`},
		{name: "concatenation binds looser than arithmetic", where: "a || b + 1 = 'x'", want: `(("a" || ("b" + 1)) = 'x')`},
		{name: "unary signs", where: "a = - -1 and b = +2", want: `(("a" = (-((-(1))))) AND ("b" = (+(2))))`},
		{name: "cast binds tighter than unary minus", where: "-a::int = 1", want: `((-(("a"::int))) = 1)`},
		{name: "chained casts", where: "a::text::varchar(10) = 'x'", want: `((("a"::text)::varchar(10)) = 'x')`},
		{name: "multi-word types", where: "a::double precision = 1 and b::character varying = 'x' and c::character(2) = 'y'",
			want: `(((("a"::double precision) = 1) AND (("b"::character varying) = 'x')) AND (("c"::character(2)) = 'y'))`},
		{name: "types with time zone", where: "a::timestamp with time zone = b::time(3) without time zone",
			want: `(("a"::timestamp with time zone) = ("b"::time(3) without time zone))`},
		{name: "timestamp without zone suffix", where: "a::timestamp = b::time", want: `(("a"::timestamp) = ("b"::time))`},
		{name: "numeric precision", where: "a::numeric(10, 2) = 1", want: `(("a"::numeric(10,2)) = 1)`},
		{name: "cast call", where: "CAST(a AS bigint) = 1", want: `(CAST("a" AS bigint) = 1)`},
		{name: "extract", where: "extract(year from created_at) = 2024", want: `(EXTRACT(YEAR FROM "created_at") = 2024)`},
		{name: "typed literals", where: "ts > now() - interval '1 day' and d >= date '2024-01-01' and t < timestamptz '2024-01-01 00:00Z'",
			want: `((("ts" > (now() - INTERVAL '1 day')) AND ("d" >= DATE '2024-01-01')) AND ("t" < TIMESTAMPTZ '2024-01-01 00:00Z'))`},
		{name: "current time keywords", where: "ts < current_timestamp and d <= CURRENT_DATE", want: `(("ts" < CURRENT_TIMESTAMP) AND ("d" <= CURRENT_DATE))`},
		{name: "functions", where: "lower(name) = 'x' and date_trunc('day', ts) = to_timestamp(0) and coalesce(a, b, 0) = 1",
			want: `(((lower("name") = 'x') AND (date_trunc('day', "ts") = to_timestamp(0))) AND (coalesce("a", "b", 0) = 1))`},
		{name: "function name is case-insensitive", where: "UPPER(a) = 'X'", want: `(upper("a") = 'X')`},
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
