package postgres

import (
	"testing"

	"hump/internal/port/database"
)

func TestSuggestWhere(t *testing.T) {
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
			got := suggestWhere(tt.columns)
			if got != tt.want {
				t.Fatalf("suggestWhere() = %q, want %q", got, tt.want)
			}

			if got == "" {
				return
			}

			if _, err := parseWhere(got); err != nil {
				t.Errorf("parseWhere(%q) error = %v, want the suggestion to be an accepted filter", got, err)
			}
		})
	}
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
