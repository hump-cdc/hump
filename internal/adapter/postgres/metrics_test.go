package postgres

import (
	"testing"

	"github.com/jackc/pglogrepl"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestUpdateReplicationLag(t *testing.T) {
	tests := []struct {
		name      string
		processed pglogrepl.LSN
		server    pglogrepl.LSN
		want      float64
	}{
		{name: "server ahead", processed: 0x100, server: 0x1000, want: 0xF00},
		{name: "processed ahead", processed: 0x1000, server: 0x100, want: 0},
		{name: "positions equal", processed: 0x1000, server: 0x1000, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMetrics()
			m.updateReplicationLag(tt.processed, tt.server)

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
