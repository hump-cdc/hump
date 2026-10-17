package funcs

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// tmplNow returns the current local time.
func tmplNow() time.Time { return time.Now() }

// asTime coerces v into a time.Time. Accepts time.Time, *time.Time, int/int64
// (interpreted as Unix seconds), or a string in RFC3339.
func asTime(v any) (time.Time, error) {
	switch x := v.(type) {
	case time.Time:
		return x, nil
	case *time.Time:
		if x == nil {
			return time.Time{}, errors.New("nil time")
		}
		return *x, nil
	case int:
		return time.Unix(int64(x), 0), nil
	case int64:
		return time.Unix(x, 0), nil
	case string:
		return time.Parse(time.RFC3339, x)
	}
	return time.Time{}, fmt.Errorf("cannot interpret %T as time", v)
}

// tmplUnixEpoch returns the Unix-second timestamp of t as a string.
func tmplUnixEpoch(v any) string {
	t, err := asTime(v)
	if err != nil {
		return ""
	}
	return strconv.FormatInt(t.Unix(), 10)
}

// tmplAgo returns the duration since v as a Go duration string.
func tmplAgo(v any) string {
	t, err := asTime(v)
	if err != nil {
		return ""
	}
	return time.Since(t).String()
}

// tmplDuration parses a seconds-count (string or int) and returns its
// Go duration representation.
func tmplDuration(v any) string {
	var secs int64
	switch x := v.(type) {
	case int:
		secs = int64(x)
	case int64:
		secs = x
	case string:
		n, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return ""
		}
		secs = n
	default:
		return ""
	}
	return (time.Duration(secs) * time.Second).String()
}

// tmplDurationRound returns the duration since v rounded down to its most
// significant unit.
func tmplDurationRound(v any) string {
	t, err := asTime(v)
	if err != nil {
		return ""
	}
	d := time.Since(t)
	if d < 0 {
		d = -d
	}
	switch {
	case d >= 365*24*time.Hour:
		return fmt.Sprintf("%dy", d/(365*24*time.Hour))
	case d >= 30*24*time.Hour:
		return fmt.Sprintf("%dmo", d/(30*24*time.Hour))
	case d >= 24*time.Hour:
		return fmt.Sprintf("%dd", d/(24*time.Hour))
	case d >= time.Hour:
		return fmt.Sprintf("%dh", d/time.Hour)
	case d >= time.Minute:
		return fmt.Sprintf("%dm", d/time.Minute)
	case d >= time.Second:
		return fmt.Sprintf("%ds", d/time.Second)
	}
	return "0s"
}

// tmplDate formats v with the given layout (Go time layout, e.g. 2006-01-02).
func tmplDate(layout string, v any) string {
	t, err := asTime(v)
	if err != nil {
		return ""
	}
	return t.Format(layout)
}

// tmplDateInZone is tmplDate but rendered in the named IANA timezone. An
// unknown zone falls back to UTC.
func tmplDateInZone(layout string, v any, zone string) string {
	t, err := asTime(v)
	if err != nil {
		return ""
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Format(layout)
}

// tmplHTMLDate formats v as YYYY-MM-DD.
func tmplHTMLDate(v any) string { return tmplDate("2006-01-02", v) }

// tmplHTMLDateInZone formats v as YYYY-MM-DD in zone.
func tmplHTMLDateInZone(v any, zone string) string {
	return tmplDateInZone("2006-01-02", v, zone)
}

// tmplToDate parses s using layout and returns the time. Parse errors abort
// rendering.
func tmplToDate(layout, s string) (time.Time, error) {
	return time.Parse(layout, s)
}

// tmplDateModify applies a Go duration string ("-1.5h", "30m") to v. Invalid
// inputs abort rendering.
func tmplDateModify(spec string, v any) (time.Time, error) {
	t, err := asTime(v)
	if err != nil {
		return time.Time{}, err
	}
	d, err := time.ParseDuration(spec)
	if err != nil {
		return time.Time{}, err
	}
	return t.Add(d), nil
}
