# adapter/sink — how this package hangs together

The delivery end of the pipeline. One row change arrives as a `port/sink.Context`, and the
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
4. Fill `capture` as its doc comment in `port/sink/capture.go` says: after the publish, on
   **both** paths, through `SetPayload`/`Set`.
5. Return through `wrapError`.

Shared conventions inside that: headers land on the capture under a `header.` prefix, the
other `Metadata` keys are unprefixed (`method`, `status`, `key`, `stream`, `sequence`,
`routingKey`) — everything sink-specific goes into that one map, so a new sink type never needs
the port widened; an empty rendered header is dropped rather than sent, and `template.ErrRender` is
permanent in every sink (`pkg/template` wraps every execution failure with it).

**Adding a sink** touches six places: a `SinkType<Name>` constant and the `SinkTypes` list in
`internal/port/sink/sink_type.go` (which is what drives `getSinkType`, so the YAML
discriminator needs no further edit), a `config/sink/config_<name>.go` implementing
`Definition` (`Type`, `DisplayName`, `Metadata`) plus `CrossValidate` and `UnmarshalYAML`, a
case in `unmarshalSink` and an entry in `crossValidators` (both `config/sink/config_sinks.go`),
a case in `sink.go`, and the subpackage above.

`Definition.Metadata` renders the `(host, target)` pair that becomes the Prometheus labels;
its contract is in `port/sink/AGENTS.md`.

## Diagnostics

HTTP request build, render and transport errors use controlled messages; status errors include
only the status code and a `diagnostic.Endpoint` label. NATS, RabbitMQ and RabbitMQ Streams
wrap delivery and lifecycle errors with `diagnostic.Error`, retaining causes for classification
without formatting their text. Their lifecycle logs and shutdown hook names use safe endpoint
labels (`Endpoints` for NATS server lists), not raw URLs.

`rabbitmq_stream/logger.go` installs a dependency logger that emits controlled events at the
original Info, Error, Debug or Warn level; it never formats the client's free-form text or
arguments. These rules do not change connection values, cache keys, captures, payloads or headers.

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
sleeps 2.1–3s (2s base + 100ms–1s jitter), so the sleep gates nothing: the next `Process` recreates the producer immediately.

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
- **RabbitMQ Streams** has no invalidation, so a sticky success outlives
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

None of these tests needs a real broker, and none adds a test dependency (no kfake, nats-server
or testcontainers):

- **HTTP** runs against `httptest.NewServer`; `http/http_test.go` also checks safe error
  serialization at Info and Debug levels and preserved error causes.
- **NATS, RabbitMQ, RabbitMQ Streams** each have a `fake_server_test.go`: a small loopback
  server that speaks just enough of the wire protocol for the real client library. It is
  stdlib-only, except that the RabbitMQ fake takes reply codes from `amqp091-go`. In the two
  RabbitMQ fakes, hooks set each declare and publish outcome and `dropClients` simulates a lost
  connection. `nats/reconciler_test.go` also uses `fakeJS`, a stub of the `jetstream.JetStream`
  interface.
- **Kafka** uses a cached client that is already closed (produce fails with
  `kgo.ErrClientClosed`) and a seed on a closed local port (produce runs into the deadline). A
  successful produce is not tested, because that needs a broker.
- `shutdown.Shutdown` runs only once per process (`sync.Once` in the global registry). The NATS,
  RabbitMQ and RabbitMQ Streams shutdown tests run in Info/Debug child processes to isolate
  logger initialization and the registry, forwarding `-test.gocoverdir` when set. They check
  lifecycle/shutdown output for secret sentinels in `nats/nats_test.go`,
  `rabbitmq/rabbitmq_test.go` and `rabbitmq_stream/rabbitmq_stream_test.go`; Streams also checks
  dependency logger levels. Kafka keeps one dedicated shutdown test.
- `sink_test.go` covers the dispatch over definition types.

`rabbitmq/returns_test.go` builds a `rabbitConn` with hand-made confirm/return channels to test
the publish-outcome logic directly. One case asserts the ack path finishes in under 5ms. It
guards against the return of a removed 50ms straggler wait; it is not a performance target.
`dev/docker-compose.yml` at the repo root brings up Kafka, NATS and RabbitMQ for manual runs.

`newTestConn` sits above `TestAwaitPublishResult`, not below the last test as the `go` skill
puts helpers.
