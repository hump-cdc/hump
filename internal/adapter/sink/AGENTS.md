# adapter/sink — how this package hangs together

The delivery end of the pipeline. One row change arrives as a `port/sink.SinkContext`, and the
adapter renders it into a broker-specific message, publishes it, and reports an error the retry
layer can classify. Everything here is called from `process_operation.binding`, once per
attempt, on a worker goroutine.

## Layout

| File | Role |
|---|---|
| `sink.go` | the `Sink` port implementation: a type switch over `config/sink` definitions |
| `kafka/` | franz-go producer, `ProduceSync` |
| `nats/` | Core NATS and JetStream, one client for both |
| `http/` | `net/http` request per row |
| `rabbitmq/` | classic AMQP 0.9.1, publisher confirms |
| `rabbitmq_stream/` | RabbitMQ Streams, one producer per stream |

Every subpackage has the same shape, and a new one should keep it:

| File | Role |
|---|---|
| `<name>.go` | `Process` (`ProcessCore` / `ProcessJetStream` in `nats`) — the whole delivery, top to bottom |
| `client.go` | the process-global connection cache (`http` has none) |
| `message.go` | template rendering into the broker's message type |
| `errors.go` | `wrapError` plus the permanent/transient classification |
| `reconciler.go` | stream/exchange declaration, where the broker has one |
| `logger.go` | `log.Scope(...)` (`http` has none) |

## The contract every adapter follows

`Process(ctx, config, operationContext, capture)` (NATS: `ProcessCore` and `ProcessJetStream`, same
signature):

1. `ctx.Err()` first, returned raw — a cancelled context is not a sink error.
2. Render, acquire a cached client, build the message. The order differs per sink (RabbitMQ
   Streams needs the rendered stream name to find its producer; RabbitMQ classic waits on
   `readyChan` before it renders), but every rendered field goes through `template.Template`.
3. Publish **synchronously**. The application layer is what retries, times out and counts, so
   an adapter never retries internally and never outlives the call.
4. Fill `capture` — after the publish, on **both** the success and the error path. A failed
   delivery is the one whose payload a reader wants. `capture` is nil unless a client is
   watching the operations log; its buffers belong to the caller and are reused, so write
   through `SetPayload`/`Set`, never replace the fields.
5. Return through `wrapError`.

Shared conventions inside that: headers land on the capture under a `header.` prefix, an
empty rendered header is dropped rather than sent, and `template.ErrRender` is permanent in
every sink (`pkg/template` wraps every execution failure with it).

**Adding a sink** touches six places: a `SinkType` constant and the `SinkTypes` list in
`internal/port/sink/sink_type.go` (which is what drives `getSinkType`, so the YAML
discriminator needs no further edit), a `config/sink/config_<name>.go` implementing
`Definition` (`Type`, `DisplayName`, `Metadata`) plus `CrossValidate` and `UnmarshalYAML`, a
case in `unmarshalSink` and an entry in `crossValidators` (both `config/sink/config_sinks.go`),
a case in `sink.go`, and the subpackage above.

`Definition.Metadata` renders the `(host, target)` pair that becomes the Prometheus labels —
see `adapter/metrics_collector/AGENTS.md` for why those strings have to match byte for byte.

## `ErrSinkPermanent` is a marker nobody reads yet

Every `wrapError` joins `sink.ErrSinkPermanent` onto errors it judges non-retriable, and the
NATS auth path marks it inline. **Nothing consumes it.** `resilience.WithRetry` retries on any
non-nil error up to `retry.maxAttempts`, and no other package references the sentinel. So the
classification is currently descriptive — it reaches the log line and the operations log entry,
not the retry decision. Keep classifying correctly; just do not assume a permanent error stops
the loop today.

## Client caches are process-global and outlive the config

Each subpackage except `http` (one shared `defaultClient`) keeps a package-level map guarded by a
package-level mutex, keyed by the
**rendered** connection fields: Kafka by url + clientID + credentials + sasl + tls, NATS by url
+ name + whichever auth group applies, RabbitMQ by url alone (auth lives in the URL), RabbitMQ
Streams by url for the environment and by (url, stream) for producers. Connection fields are
ordinary templates with no static-only validation, so a url that renders per row mints — and
keeps — one client per distinct value.

Eviction is uneven: Kafka and NATS never delete an entry (the `shutdown.StageSinks` hook closes
the client and leaves the map alone), RabbitMQ deletes its entry from inside the shutdown hook,
and RabbitMQ Streams also evicts a producer from
`watchProducerClose` when the broker closes it. That goroutine deletes the entry and *then*
sleeps ~2s, so the sleep gates nothing: the next `Process` recreates the producer immediately.

## Three reconcilers, one shape

JetStream streams, AMQP exchanges and RabbitMQ streams are all declared lazily on first
publish, by a reconciler with the same structure: a `map[name]*state`, a per-name mutex so one
publisher declares at a time, and a sticky `done`/`err` replayed to later callers. A transient
transport error is returned **without** setting `done`, so a cold broker is retried instead of
being cached as a permanent failure. The differences:

- **NATS** skips entirely when `stream.subjects` is empty, and has no invalidation — a
  reconnect reuses the same `nats.Conn`, so the state stays valid.
- **RabbitMQ classic** invalidates on connection reconnect only. Reopening a dead channel
  (`reopenChannelLocked`) deliberately does not, because exchange declarations live on the
  broker, not on the channel.
- **RabbitMQ Streams** has an `invalidate` that is never called, so a sticky success outlives
  the stream on the broker: if the stream is dropped, the evicted producer is recreated against
  the cached outcome rather than re-declared, even in `create-when-missing` mode.

## Per-sink things that will bite

- **Kafka.** `UnknownTopicOrPartition` is classified permanent, against Kafka's own retriable
  flag: in a producer with no auto-create it means a missing topic. Broker health is logged off
  *connect* outcomes, not disconnects — `hooks.go` explains why at length.
- **HTTP.** `defaultClient` has no timeout of its own; the bound comes from
  `resilience.WithTimeout` and the request context. 4xx is permanent except 408 and 429, 5xx
  stays transient, an unbuildable request and a `DNSError.IsNotFound` are permanent. The
  response body is drained into `io.Discard` so the connection can be reused.
- **NATS.** Auth precedence is credentials → nkey seed → user/password → token, first match
  wins; failures there mark `ErrSinkPermanent` inline, because `isPermanentNatsError` would not
  catch them. Subjects are validated before publish (no wildcards, no empty segments). A Core
  publish is followed by a flush bounded by the remaining context deadline, 5s if there is
  none.
  `Nats-Expected-Stream` is always overwritten with the configured stream, and the mismatch it
  produces (API code 10060) is checked *before* the permanent list so it stays retriable.
- **RabbitMQ classic.** One connection, one confirm-mode channel, one publish in flight —
  `rc.mu` is held across the publish *and* the confirm/return drain, which is what makes the
  next item on each channel attributable to this publish. **Read `returns.go` before touching
  the publish path**; it documents the return-before-ack ordering the fast path relies on.
  `mandatory` defaults to true, so an unroutable message is a permanent error rather than a
  silent drop. A supervisor goroutine reconnects with 2s + jitter and swaps `ready` so
  `Process` blocks instead of publishing into a dead channel.
- **RabbitMQ Streams.** `NotifyPublishConfirmation` *replaces* the producer's confirmation
  channel on every call, so it may only be called while `publishMu` is held — that mutex is the
  reason a producer publishes one message at a time.

## Tests

`rabbitmq/returns_test.go` is the only test here: it builds a `rabbitConn` with hand-made
confirm/return channels, so the publish-outcome logic is exercised without a broker. One case
asserts the ack path finishes in under 5ms — a regression lock on a removed 50ms straggler
wait, not a performance nicety. There are no broker integration tests here;
`dev/docker-compose.yml` brings up Kafka, NATS and RabbitMQ for manual runs.

Helpers live below the last test function — see the `go` skill.
