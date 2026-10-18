package kafka

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// brokerState is the last observed health of a broker, tracked per NodeID so
// benign churn can be told apart from real unavailability.
type brokerState int

const (
	// stateUnknown means no connect attempt has resolved yet.
	stateUnknown brokerState = iota
	// stateConnected means the last connect attempt succeeded.
	stateConnected
	// stateFailed means the last connect attempt errored (broker unreachable).
	stateFailed
)

// connHooks tracks per-broker connection state to separate benign connection
// churn from real outages. franz-go closes idle connections (and opens new ones
// per produce/fetch/group/slow/normal cxn slot), so disconnects fire routinely
// with no error attached and on their own say nothing about broker health. The
// reliable signal is the *next connect*: an idle reap reconnects instantly,
// while an unavailable broker fails to connect. So disconnects are logged only
// at debug level, and Warn/Info are emitted off connect outcome transitions.
type connHooks struct {
	url string

	mu    sync.Mutex
	state map[int32]brokerState
}

var (
	_ kgo.HookBrokerConnect    = (*connHooks)(nil)
	_ kgo.HookBrokerDisconnect = (*connHooks)(nil)
	_ kgo.HookClientClosed     = (*connHooks)(nil)
)

// newConnHooks returns hooks logging broker connection lifecycle transitions
// for the client created against url.
func newConnHooks(url string) *connHooks {
	return &connHooks{
		url:   url,
		state: map[int32]brokerState{},
	}
}

// OnBrokerConnect logs connect-outcome transitions, which are the only reliable
// signal of broker health: a Warn on the first failed dial (a real outage), an
// Info on the first successful connect and on recovery from a failed state.
// A success that merely follows a benign idle reap (prior state still connected)
// is silent.
func (h *connHooks) OnBrokerConnect(meta kgo.BrokerMetadata, _ time.Duration, _ net.Conn, err error) {
	h.mu.Lock()
	prev := h.state[meta.NodeID]
	if err == nil {
		h.state[meta.NodeID] = stateConnected
	} else {
		h.state[meta.NodeID] = stateFailed
	}
	h.mu.Unlock()

	broker := brokerString(meta)

	if err != nil {
		if prev != stateFailed {
			logger.Warn("Kafka broker unreachable", "seed", h.url, "broker", broker, "err", err)
		}
		return
	}

	switch prev {
	case stateUnknown:
		logger.Info("Kafka broker connected", "seed", h.url, "broker", broker)
	case stateFailed:
		logger.Info("Kafka broker reconnected", "seed", h.url, "broker", broker)
	case stateConnected:
		// Already connected: a routine reconnect after an idle reap stays silent.
	}
}

// OnBrokerDisconnect logs at debug level only. A disconnect carries no error and
// fires routinely when franz-go reaps idle connections, so it says nothing about
// broker health on its own — the following connect attempt does. Broker state is
// intentionally left unchanged so a benign reap+reconnect stays silent while a
// disconnect followed by a failed reconnect surfaces as unreachable.
func (h *connHooks) OnBrokerDisconnect(meta kgo.BrokerMetadata, _ net.Conn) {
	logger.Debug("Kafka broker connection closed", "seed", h.url, "broker", brokerString(meta))
}

// OnClientClosed fires once when the client is shut down. Mirrors the
// NATS ClosedHandler — useful as a "we are done" marker in logs.
func (h *connHooks) OnClientClosed(_ *kgo.Client) {
	logger.Info("Kafka client closed", "seed", h.url)
}

// brokerString renders a broker as "host:port" (NodeID for seed brokers, which
// franz-go assigns very negative IDs).
func brokerString(meta kgo.BrokerMetadata) string {
	if meta.NodeID < 0 {
		return fmt.Sprintf("seed:%s:%d", meta.Host, meta.Port)
	}
	return fmt.Sprintf("%s:%d", meta.Host, meta.Port)
}
