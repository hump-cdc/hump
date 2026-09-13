package metricscollector

import (
	"errors"
	"testing"
	"time"
)

func TestDurationResolvesSupportedWidths(t *testing.T) {
	want := map[Duration]time.Duration{
		Duration1m:  time.Minute,
		Duration5m:  5 * time.Minute,
		Duration10m: 10 * time.Minute,
	}

	for _, d := range Durations {
		got, err := d.Duration()
		if err != nil || got != want[d] {
			t.Fatalf("got %v, %v for %q, want %v", got, err, d, want[d])
		}
	}
}

func TestDurationRejectsUnknownWidth(t *testing.T) {
	if _, err := Duration("2m").Duration(); !errors.Is(err, ErrUnknownDuration) {
		t.Fatalf("got %v, want ErrUnknownDuration", err)
	}
}
