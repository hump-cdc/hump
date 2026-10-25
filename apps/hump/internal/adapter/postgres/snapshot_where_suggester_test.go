package postgres

import (
	"strings"
	"testing"

	"hump/internal/port/database"
)

func TestSnapshotWhereSuggesterSuggest(t *testing.T) {
	tests := []struct {
		name    string
		columns []tablesColumn
		want    string
	}{
		{name: "no columns", want: ""},
		{
			name:    "no temporal column",
			columns: []tablesColumn{column("id", "bigint"), column("updated_at", "text")},
			want:    "",
		},
		{
			name:    "time key wins over names",
			columns: []tablesColumn{column("updated_at", "timestamptz"), timeKey(column("ts_bucket", "timestamp with time zone"))},
			want:    `"ts_bucket" >= now() - interval '7 days'`,
		},
		{
			name:    "integer time key is ignored",
			columns: []tablesColumn{timeKey(column("epoch", "bigint")), column("created_at", "date")},
			want:    `"created_at" >= now() - interval '7 days'`,
		},
		{
			name:    "updated wins over created",
			columns: []tablesColumn{column("created_at", "timestamp with time zone"), column("updated_at", "timestamp with time zone")},
			want:    `"updated_at" >= now() - interval '7 days'`,
		},
		{
			name:    "created wins over event time",
			columns: []tablesColumn{column("ts", "timestamp without time zone"), column("CreatedAt", "timestamp(3) without time zone")},
			want:    `"CreatedAt" >= now() - interval '7 days'`,
		},
		{
			name:    "event time",
			columns: []tablesColumn{column("id", "bigint"), column("occurred_at", "timestamp with time zone")},
			want:    `"occurred_at" >= now() - interval '7 days'`,
		},
		{
			name:    "names normalize case and separators",
			columns: []tablesColumn{column("Last-Modified", "timestamp with time zone")},
			want:    `"Last-Modified" >= now() - interval '7 days'`,
		},
		{
			name:    "indexed column wins a tie",
			columns: []tablesColumn{column("updated", "timestamp with time zone"), indexed(column("modified_at", "timestamp with time zone"))},
			want:    `"modified_at" >= now() - interval '7 days'`,
		},
		{
			name:    "earlier column wins a full tie",
			columns: []tablesColumn{column("updated", "timestamp with time zone"), column("modified_at", "timestamp with time zone")},
			want:    `"updated" >= now() - interval '7 days'`,
		},
		{
			name:    "sole now default",
			columns: []tablesColumn{column("id", "bigint"), nowDefault(column("registered", "timestamp with time zone"))},
			want:    `"registered" >= now() - interval '7 days'`,
		},
		{
			name: "several now defaults are ambiguous",
			columns: []tablesColumn{
				nowDefault(column("registered", "timestamp with time zone")),
				nowDefault(column("confirmed", "timestamp with time zone")),
			},
			want: "",
		},
		{
			name:    "unrecognized temporal column",
			columns: []tablesColumn{column("deleted_at", "timestamp with time zone"), column("birthday", "date")},
			want:    "",
		},
		{
			name:    "name needing quotes",
			columns: []tablesColumn{column(`up"dated`, "timestamp with time zone"), timeKey(column(`Odd "Time"`, "timestamp with time zone"))},
			want:    `"Odd ""Time""" >= now() - interval '7 days'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newSnapshotWhereSuggester(tt.columns).Suggest()
			if got != tt.want {
				t.Fatalf("Suggest() = %q, want %q", got, tt.want)
			}

			if got == "" {
				return
			}

			if _, err := newSnapshotWhereParser().Parse(got); err != nil {
				t.Errorf("Parse(%q) error = %v, want the suggestion to be an accepted filter", got, err)
			}
		})
	}
}

func FuzzSnapshotWhereSuggester(f *testing.F) {
	f.Add("updated_at", "timestamp with time zone", false, false, false)
	f.Add("ts_bucket", "timestamptz", true, true, false)
	f.Add("created", "date", false, false, true)
	f.Add("User\"Id", "timestamp(3) without time zone", true, false, false)
	f.Add("a -- b", "timestamp", true, false, false)
	f.Add("x'; DROP TABLE t; --", "timestamp", false, false, true)

	f.Fuzz(func(t *testing.T, name, typ string, isTimeKey, isIndexed, isNowDefault bool) {
		if name == "" || strings.ContainsRune(name, 0) {
			t.Skip("PostgreSQL identifiers are never empty and never hold NUL")
		}

		candidate := tablesColumn{
			TableColumn: database.TableColumn{Name: name, Type: typ},
			TimeKey:     isTimeKey,
			Indexed:     isIndexed,
			NowDefault:  isNowDefault,
		}

		got := newSnapshotWhereSuggester([]tablesColumn{column("id", "bigint"), candidate}).Suggest()
		if got == "" {
			return
		}

		rendered, err := newSnapshotWhereParser().Parse(got)
		if err != nil {
			t.Fatalf("Suggest() = %q, which the filter parser rejects: %v", got, err)
		}

		if !strings.Contains(rendered, quoteIdentifier(name)) {
			t.Fatalf("Parse(%q) = %q, want it to filter on %s", got, rendered, quoteIdentifier(name))
		}
	})
}

// column is a plain column of the given name and type.
func column(name, typ string) tablesColumn {
	return tablesColumn{TableColumn: database.TableColumn{Name: name, Type: typ}}
}

// timeKey marks c as the table's time key.
func timeKey(c tablesColumn) tablesColumn {
	c.TimeKey = true
	return c
}

// indexed marks c as leading an index.
func indexed(c tablesColumn) tablesColumn {
	c.Indexed = true
	return c
}

// nowDefault marks c as defaulting to the current time.
func nowDefault(c tablesColumn) tablesColumn {
	c.NowDefault = true
	return c
}
