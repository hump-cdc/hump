package postgres

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestSetReplicationLag(t *testing.T) {
	tests := []struct {
		name  string
		bytes float64
		want  float64
	}{
		{name: "positive", bytes: 0xF00, want: 0xF00},
		{name: "zero", bytes: 0, want: 0},
		{name: "negative", bytes: -0x100, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMetrics()
			m.setReplicationLag(tt.bytes)

			got := testutil.ToFloat64(m.replicationLag)
			if got != tt.want {
				t.Errorf("replicationLag = %v, want %v", got, tt.want)
			}
		})
	}
}

// newTestMetrics builds a metrics set on a private registry, so that each call
// can register replication_lag_bytes without colliding with an earlier one.
func newTestMetrics() *metrics {
	return newMetrics("db", promauto.With(prometheus.NewRegistry()))
}
