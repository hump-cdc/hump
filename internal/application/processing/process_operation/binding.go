package process_operation

import (
	"context"
	"fmt"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"
	metricscollector "hump/internal/port/metrics_collector"
	operationscollector "hump/internal/port/operations_collector"
	"hump/internal/port/sink"
	"hump/pkg/app/shutdown"
	"hump/pkg/resilience"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// handler is one step of a binding's delivery chain.
type handler = resilience.Handler[*sink.Context]

type binding struct {
	sinkName              string
	table                 string
	operations            map[domain.OperationType]struct{} // operations are the row operations the sink receives.
	observer              Observer                          // observer is nil when the web UI is disabled.
	log                   OperationLog                      // log is nil when the operations log is disabled.
	conn                  metricscollector.Connection
	op                    operation
	fn                    handler
	messageCounterSuccess prometheus.Counter
	messageCounterRetry   prometheus.Counter
	messageCounterFailure prometheus.Counter
	durationHistogram     prometheus.Observer
}

// operation is the per-delivery scratch state of one binding, reused so a
// logged delivery allocates nothing. The worker pool submits a binding once per
// awaited batch from a single goroutine, so a binding never has two deliveries
// in flight; inflight guards that invariant rather than making the fields safe
// for concurrent use.
type operation struct {
	inflight atomic.Bool
	rec      operationscollector.Record
	capture  sink.Capture
}

func newBinding(
	cfg sinkcfg.Config,
	schema domain.TableSchema,
	tableName domain.TableName,
	sinksPort sink.Sink,
	sinkMessagesCounter *prometheus.CounterVec,
	sinkDurationHistogram *prometheus.HistogramVec,
	observer Observer,
	opLog OperationLog,
) *binding {
	host, target := cfg.Sink.Metadata(sink.Context{
		Schema: schema,
		Table:  tableName,
	})
	operations := make(map[domain.OperationType]struct{}, len(cfg.Operations))
	for _, op := range cfg.Operations {
		operations[op] = struct{}{}
	}

	b := &binding{
		sinkName:   string(cfg.Sink.Type()),
		table:      fmt.Sprintf("%s.%s", schema, tableName),
		operations: operations,
		observer:   observer,
		log:        opLog,
		conn: metricscollector.Connection{
			Table:    domain.TableIdentifier{Schema: schema, Name: tableName},
			SinkType: string(cfg.Sink.Type()),
			Host:     host,
			Target:   target,
		},
		messageCounterSuccess: sinkMessagesCounter.WithLabelValues(string(domain.DeliveryStatusSuccess), string(cfg.Sink.Type()), host, target),
		messageCounterRetry:   sinkMessagesCounter.WithLabelValues(string(domain.DeliveryStatusRetry), string(cfg.Sink.Type()), host, target),
		messageCounterFailure: sinkMessagesCounter.WithLabelValues(string(domain.DeliveryStatusFailure), string(cfg.Sink.Type()), host, target),
		durationHistogram:     sinkDurationHistogram.WithLabelValues(string(cfg.Sink.Type()), host, target),
	}

	// Two chains rather than one with a flag inside: the logged one writes into
	// b.op from every attempt, so the choice has to be made once, outside the
	// retry loop. plain is what the binding ran before the log existed, and is
	// what a delivery takes whenever nobody is watching.
	retry := func(fn handler) handler {
		return resilience.WithRetry(fn, uint(cfg.Retry.MaxAttempts), cfg.Retry.Backoff.Policy(), b, shutdown.Draining())
	}
	deliver := func(capture *sink.Capture) handler {
		return func(ctx context.Context, operationContext *sink.Context) error {
			return sinksPort.Process(ctx, cfg.Sink, operationContext, capture)
		}
	}

	plain := retry(resilience.WithTimeout(deliver(nil), cfg.Retry.AttemptTimeout))
	logged := retry(b.withAttempt(resilience.WithTimeout(deliver(&b.op.capture), cfg.Retry.AttemptTimeout)))

	fn := b.withOperation(plain, logged)
	fn = resilience.WithStatus(fn, b)
	fn = resilience.WithFailureMode(fn, cfg.Failure.Mode, b)
	b.fn = fn

	return b
}

// accepts reports whether the sink is configured to receive op.
func (b *binding) accepts(op domain.OperationType) bool {
	_, ok := b.operations[op]
	return ok
}

func (b *binding) Process(ctx context.Context, operationContext *sink.Context) error {
	return b.fn(ctx, operationContext)
}

// withOperation records one delivery, retries and all, as a single entry in the
// operations log. It sits outside the retry loop so it sees the terminal error,
// and inside WithStatus so the entry agrees with the counters, and so a skipped
// failure is still logged as one.
func (b *binding) withOperation(plain, logged handler) handler {
	return func(ctx context.Context, operationContext *sink.Context) error {
		if b.log == nil || !b.log.Connected() {
			return plain(ctx, operationContext)
		}

		// A second delivery on the same binding would share the scratch record,
		// so it is delivered without being logged rather than corrupting it.
		if !b.op.inflight.CompareAndSwap(false, true) {
			return plain(ctx, operationContext)
		}
		defer b.op.inflight.Store(false)

		b.op.begin(operationContext, b.conn)
		err := logged(ctx, operationContext)
		b.op.end(err)

		b.log.Record(&b.op.rec)
		b.op.rebind()

		return err
	}
}

// withAttempt appends one attempt to the open record. It wraps the timeout so a
// deadline counts as the attempt's error and its full duration is measured.
func (b *binding) withAttempt(fn handler) handler {
	return func(ctx context.Context, operationContext *sink.Context) error {
		started := time.Now()
		err := fn(ctx, operationContext)

		attempt := operationscollector.Attempt{StartedAt: started, Duration: time.Since(started)}
		if err != nil {
			attempt.Error = err.Error()
		}

		b.op.rec.Attempts = append(b.op.rec.Attempts, attempt)

		return err
	}
}

// begin resets the scratch record for one delivery over conn, keeping every
// buffer it already owns.
func (o *operation) begin(operationContext *sink.Context, conn metricscollector.Connection) {
	o.capture.Target = ""
	o.capture.Payload = o.capture.Payload[:0]
	clear(o.capture.Metadata)

	o.rec = operationscollector.Record{
		StartedAt: time.Now(),
		Timestamp: operationContext.Timestamp,
		Database:  operationContext.Database,
		Table:     conn.Table,
		Operation: operationContext.Operation,
		SinkType:  conn.SinkType,
		Host:      conn.Host,
		Target:    conn.Target,
		TargetKey: conn.Target,
		Attempts:  o.rec.Attempts[:0],
	}
}

// end closes the record with the terminal error of the retry loop and labels
// every attempt. An attempt's status is only knowable once the loop is done:
// a failed one is a retry unless it was the last.
func (o *operation) end(err error) {
	o.rec.Duration = time.Since(o.rec.StartedAt)
	o.rec.Payload = o.capture.Payload
	o.rec.Metadata = o.capture.Metadata

	if o.capture.Target != "" {
		o.rec.Target = o.capture.Target
	}

	o.rec.Status = domain.DeliveryStatusSuccess
	if err != nil {
		o.rec.Status = domain.DeliveryStatusFailure
		o.rec.Error = err.Error()
	}

	last := len(o.rec.Attempts) - 1
	for i := range o.rec.Attempts {
		switch {
		case o.rec.Attempts[i].Error == "":
			o.rec.Attempts[i].Status = domain.DeliveryStatusSuccess
		case i == last && o.rec.Status == domain.DeliveryStatusFailure:
			o.rec.Attempts[i].Status = domain.DeliveryStatusFailure
		default:
			o.rec.Attempts[i].Status = domain.DeliveryStatusRetry
		}
	}

	o.rec.Retries = max(last, 0)
}

// rebind points the capture back at the buffers the collector handed over in
// exchange for the ones it kept. Without it the next attempt would append into
// the record the log already stored.
func (o *operation) rebind() {
	o.capture.Payload = o.rec.Payload[:0]
	o.capture.Metadata = o.rec.Metadata
}

// OnDuration reports how long one delivery attempt took.
func (b *binding) OnDuration(elapsed time.Duration) {
	b.durationHistogram.Observe(elapsed.Seconds())
}

// OnSuccess reports a delivery that finished without an error.
func (b *binding) OnSuccess() {
	b.messageCounterSuccess.Add(1)
	b.record(domain.DeliveryStatusSuccess)
}

// OnRetry reports that a failed attempt is about to be repeated.
func (b *binding) OnRetry() {
	b.messageCounterRetry.Add(1)
	b.record(domain.DeliveryStatusRetry)
}

// OnFailure reports a delivery that exhausted its attempts.
func (b *binding) OnFailure() {
	b.messageCounterFailure.Add(1)
	b.record(domain.DeliveryStatusFailure)
}

// OnSkip reports a failed delivery dropped by the sink's skip failure mode.
func (b *binding) OnSkip(err error) {
	logger.Error("Skipping failed delivery", "sink", b.sinkName, "table", b.table, "error", err)
}

// record reports one delivery with status to the observer. The sink metrics
// carry no table, so the dashboard reads its per-connection counts from here.
func (b *binding) record(status domain.DeliveryStatus) {
	if b.observer == nil {
		return
	}

	b.observer.Record(b.conn, status, 1)
}
