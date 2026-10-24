package funcs

import (
	"strings"
	"testing"
	"time"
)

func TestTimeTemplates_Date(t *testing.T) {
	tt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := []struct {
		name string
		src  string
		data any
		want string
	}{
		{"date layout", `{{ date "2006-01-02" .Value }}`, V(tt), "2024-01-02"},
		{"date from rfc3339 string", `{{ date "2006-01-02" .Value }}`, V("2024-01-02T03:04:05Z"), "2024-01-02"},
		{"date from unix int", `{{ date "2006-01-02" .Value }}`, V(0), "1970-01-01"},
		{"date bad input", `{{ date "2006-01-02" .Value }}`, V("bad"), ""},
		{"htmldate", `{{ htmldate .Value }}`, V(tt), "2024-01-02"},
		{"dateinzone NY", `{{ dateinzone "2006-01-02 15:04" .Value "America/New_York" }}`,
			V(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)), "2024-01-01 19:00"},
		{"dateinzone unknown zone falls back to UTC", `{{ dateinzone "2006-01-02" .Value "Not/AZone" }}`,
			V(tt), "2024-01-02"},
		{"dateinzone bad time", `{{ dateinzone "2006" .Value "UTC" }}`, V("bad"), ""},
		{"htmldateinzone", `{{ htmldateinzone .Value "UTC" }}`, V(tt), "2024-01-02"},
		{"unixepoch from time", `{{ unixepoch .Value }}`, V(tt), "1704164645"},
		{"unixepoch from int64", `{{ unixepoch .Value }}`, V(int64(1704164645)), "1704164645"},
		{"unixepoch bad", `{{ unixepoch .Value }}`, V("bad"), ""},
		{"duration int seconds", `{{ duration .Value }}`, V(90), "1m30s"},
		{"duration int64 seconds", `{{ duration .Value }}`, V(int64(90)), "1m30s"},
		{"duration string seconds", `{{ duration .Value }}`, V("60"), "1m0s"},
		{"duration bad string", `{{ duration .Value }}`, V("bad"), ""},
		{"duration unsupported", `{{ duration .Value }}`, V(true), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustRender(t, tc.src, tc.data)
			if got != tc.want {
				t.Fatalf("%s = %q, want %q", tc.src, got, tc.want)
			}
		})
	}
}

func TestTimeTemplates_Now(t *testing.T) {
	got := mustRender(t, `{{ now | date "2006" }}`, nil)
	if len(got) != 4 {
		t.Fatalf("now date = %q", got)
	}
}

func TestTimeTemplates_Ago(t *testing.T) {
	past := time.Now().Add(-2 * time.Second)
	got := mustRender(t, `{{ ago .Value }}`, V(past))
	if got == "" {
		t.Fatalf("ago empty")
	}
	if got := mustRender(t, `{{ ago .Value }}`, V("bad")); got != "" {
		t.Fatalf("bad input = %q", got)
	}
}

func TestTimeTemplates_DurationRound(t *testing.T) {
	now := time.Now()
	cases := []struct {
		offset time.Duration
		suffix string
	}{
		{-2 * time.Second, "s"},
		{-3 * time.Minute, "m"},
		{-2 * time.Hour, "h"},
		{-3 * 24 * time.Hour, "d"},
		{-40 * 24 * time.Hour, "mo"},
		{-400 * 24 * time.Hour, "y"},
		// Future time => negative since, exercising the d < 0 branch.
		{2 * time.Hour, "h"},
	}
	for _, tc := range cases {
		got := mustRender(t, `{{ durationround .Value }}`, V(now.Add(tc.offset)))
		if !strings.HasSuffix(got, tc.suffix) {
			t.Fatalf("offset %v: got %q, want suffix %q", tc.offset, got, tc.suffix)
		}
	}
	if got := mustRender(t, `{{ durationround .Value }}`, V("bad")); got != "" {
		t.Fatalf("bad = %q", got)
	}
}

func TestTimeTemplates_ToDate(t *testing.T) {
	got := mustRender(t, `{{ (todate "2006-01-02" .Value).Year }}`, V("2024-01-02"))
	if got != "2024" {
		t.Fatalf("got %q", got)
	}
	if _, err := render(t, `{{ todate "2006-01-02" .Value }}`, V("bad")); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestTimeTemplates_DateModify(t *testing.T) {
	tt := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)
	got := mustRender(t, `{{ (datemodify "1h30m" .Value).Hour }}`, V(tt))
	if got != "13" {
		t.Fatalf("hour = %q", got)
	}
	if _, err := render(t, `{{ datemodify "bad" .Value }}`, V(tt)); err == nil {
		t.Fatalf("expected duration parse error")
	}
	if _, err := render(t, `{{ datemodify "1h" .Value }}`, V("bad")); err == nil {
		t.Fatalf("expected time parse error")
	}
}

func TestTimeTemplates_AsTimeInputs(t *testing.T) {
	tt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := []struct {
		name string
		v    any
		want string
	}{
		{"time pointer", &tt, "1704164645"},
		{"nil time pointer", (*time.Time)(nil), ""},
		{"unsupported type", 1.5, ""},
	}
	for _, tc := range cases {
		if got := mustRender(t, `{{ unixepoch .Value }}`, V(tc.v)); got != tc.want {
			t.Errorf("unixepoch(%s) = %q, want %q", tc.name, got, tc.want)
		}
	}
}
