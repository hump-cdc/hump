# RabbitMQ Sink — PRD

## 1. Overview

Add two new sink types to hump that publish WAL operations to RabbitMQ:

- **`rabbitmq`** — classic AMQP 0.9.1 over the `github.com/rabbitmq/amqp091-go` client.
- **`rabbitmq-stream`** — RabbitMQ Streams over the `github.com/rabbitmq/rabbitmq-stream-go-client` client.

Both follow the patterns established by `internal/sinks/kafka`, `internal/sinks/http`, and `internal/sinks/nats`:
client caching by rendered connection parameters, `shutdown.OnShutdown` cleanup, template-rendered config, transient-vs-permanent error classification via `wrapError` joining `errs.ErrSinkPermanent`, and `Process(ctx, cfg, opCtx)` entrypoints called by the existing retry/delivery layers.

The framework's `Retry`, `Delivery`, and `Failure` types are not touched.

---

## 2. File layout

```
internal/sinks/rabbitmq/                 # classic AMQP sink
    rabbitmq.go        # Process(ctx, cfg, opCtx) entrypoint
    client.go          # connection cache, reconnect loop, channel management
    message.go         # buildMessage: routingKey, payload, basic.Properties, headers
    reconciler.go      # exchange create-when-missing (per (connection, exchange))
    returns.go         # NotifyReturn handling (correlated to in-flight publish)
    errors.go          # wrapError, isTransient/isPermanent classification
    logger.go          # log.Scope("rabbitmq")

internal/sinks/rabbitmq_stream/          # stream sink
    rabbitmq_stream.go # Process(ctx, cfg, opCtx) entrypoint
    client.go          # environment + producer cache, reconnect
    message.go         # buildMessage: payload, application properties (headers)
    reconciler.go      # stream passive|create-when-missing
    errors.go          # wrapError, classification
    logger.go          # log.Scope("rabbitmq-stream")
```

Default templates added under `internal/types/config/templ/`:

- `rabbitmq_routing_key.tmpl` → `{{ .Schema }}.{{ .Table }}`
- `rabbitmq_exchange.tmpl` (used as default `exchange` if unset) → `hump.{{ .Schema }}`
- `rabbitmq_stream_name.tmpl` → `hump.{{ .Schema }}.{{ .Table }}`

Existing `payload.tmpl`, `idempotency_header.tmpl`, `traceparent_header.tmpl` are reused.

Two new `SinkType` constants in `internal/types/sink/sink.go`:

```go
SinkTypeRabbitMQ       SinkType = "rabbitmq"
SinkTypeRabbitMQStream SinkType = "rabbitmq-stream"
```

`SinkTypes` slice and the discriminator switch in `internal/types/config/config_sink_base.go` updated accordingly.

---

## 3. Configuration schema

### 3.1 Classic AMQP sink (`rabbitmq`)

```yaml
- name: rabbitmq1
  rabbitmq:
      url: "amqp://hump:hump@localhost:5672/"        # required; amqps:// for TLS
      exchange:
          name: "hump.{{.Schema}}"                   # templated; default "hump.{{.Schema}}"
          type: topic                                # one of: direct|topic|fanout|headers; default topic
          durable: true                              # default true
          autoDelete: false                          # default false
          mode: create-when-missing                  # one of: passive|create-when-missing; default create-when-missing
          arguments: {}                              # optional map[string]any for x-* args; default nil
      routingKey: "{{.Schema}}.{{.Table}}"           # templated; default "{{.Schema}}.{{.Table}}"
      mandatory: true                                # default true
      deliveryMode: persistent                       # one of: persistent|transient; default persistent
      payload: "<json>"                              # templated; default payload.tmpl
      properties:                                    # all sub-fields are templated, optional
          messageId: "{{ sha256 ... }}"              # default = idempotency_header.tmpl
          correlationId: ""
          type: ""
          appId: "hump"
          userId: ""                                 # broker validates against connection user when set
          contentType: "application/json"            # default "application/json"
          contentEncoding: ""
          expiration: ""                             # AMQP per-message TTL in ms, as string
          priority: ""                               # 0..255 as string; rendered & parsed
      headers:                                       # templated map; default {X-Idempotency-Key, traceparent}
          X-Idempotency-Key: "{{ sha256 ... }}"
          traceparent: '{{index .Headers "traceparent"}}'
```

#### Config struct sketch

```go
type RabbitMQExchange struct {
    Name       *template.Template            `yaml:"name,omitempty"`
    Type       string                        `yaml:"type,omitempty"       validate:"omitempty,oneof=direct topic fanout headers"`
    Durable    bool                          `yaml:"durable,omitempty"`
    AutoDelete bool                          `yaml:"autoDelete,omitempty"`
    Mode       RabbitMQExchangeMode          `yaml:"mode,omitempty"       validate:"omitempty,oneof=passive create-when-missing"`
    Arguments  map[string]any                `yaml:"arguments,omitempty"`
}

type RabbitMQProperties struct {
    MessageID       *template.Template `yaml:"messageId,omitempty"`
    CorrelationID   *template.Template `yaml:"correlationId,omitempty"`
    Type            *template.Template `yaml:"type,omitempty"`
    AppID           *template.Template `yaml:"appId,omitempty"`
    UserID          *template.Template `yaml:"userId,omitempty"`
    ContentType     *template.Template `yaml:"contentType,omitempty"`
    ContentEncoding *template.Template `yaml:"contentEncoding,omitempty"`
    Expiration      *template.Template `yaml:"expiration,omitempty"`
    Priority        *template.Template `yaml:"priority,omitempty"`
}

type RabbitMQSink struct {
    URL *template.Template `yaml:"url" validate:"required"`

    Exchange     RabbitMQExchange              `yaml:"exchange,omitempty"`
    RoutingKey   *template.Template            `yaml:"routingKey,omitempty"`
    Mandatory    *bool                         `yaml:"mandatory,omitempty"`        // pointer so unset → default true
    DeliveryMode RabbitMQDeliveryMode          `yaml:"deliveryMode,omitempty"      validate:"omitempty,oneof=persistent transient"`
    Payload      *template.Template            `yaml:"payload,omitempty"`
    Properties   RabbitMQProperties            `yaml:"properties,omitempty"`
    Headers      map[string]*template.Template `yaml:"headers,omitempty"`
}
```

`UnmarshalYAML` calls `setDefaults()` to fill: `Exchange.Name` → default template, `Exchange.Type` → `topic`, `Exchange.Durable` → `true`, `Exchange.Mode` → `create-when-missing`, `RoutingKey` → default template, `Mandatory` → `&true`, `DeliveryMode` → `persistent`, `Payload` → default, `Properties.MessageID` → idempotency template, `Properties.ContentType` → `application/json`, `Headers` → `{X-Idempotency-Key, traceparent}` if absent.

### 3.2 Stream sink (`rabbitmq-stream`)

```yaml
- name: rabbitmq-stream
  rabbitmq-stream:
      url: "rabbitmq-stream://hump:hump@localhost:5552/%2F"   # required; rabbitmq-stream+tls:// for TLS
      stream:
          name: "hump.{{.Schema}}.{{.Table}}"                 # templated; default "hump.{{.Schema}}.{{.Table}}"
          mode: create-when-missing                           # one of: passive|create-when-missing; default create-when-missing
          maxAge: 24h                                         # time.Duration; default 24h
          maxLengthBytes: 2GB                                 # parsed by units helper or raw int; default 2 GiB
          maxSegmentSizeBytes: 500MB                          # default 500 MiB
      payload: "<json>"                                       # templated; default payload.tmpl
      headers:                                                # templated map; mapped to application-properties
          X-Idempotency-Key: "{{ sha256 ... }}"
          traceparent: '{{index .Headers "traceparent"}}'
```

#### Config struct sketch

```go
type RabbitMQStreamConfig struct {
    Name                *template.Template       `yaml:"name,omitempty"`
    Mode                RabbitMQStreamMode       `yaml:"mode,omitempty"                validate:"omitempty,oneof=passive create-when-missing"`
    MaxAge              time.Duration            `yaml:"maxAge,omitempty"              validate:"omitempty,gt=0"`
    MaxLengthBytes      int64                    `yaml:"maxLengthBytes,omitempty"      validate:"omitempty,gt=0"`
    MaxSegmentSizeBytes int64                    `yaml:"maxSegmentSizeBytes,omitempty" validate:"omitempty,gt=0"`
}

type RabbitMQStreamSink struct {
    URL *template.Template `yaml:"url" validate:"required"`

    Stream  RabbitMQStreamConfig          `yaml:"stream,omitempty"`
    Payload *template.Template            `yaml:"payload,omitempty"`
    Headers map[string]*template.Template `yaml:"headers,omitempty"`
}
```

Defaults: `Stream.Name` → default template, `Stream.Mode` → `create-when-missing`, `Stream.MaxAge` → `24h`, `Stream.MaxLengthBytes` → 2 GiB, `Stream.MaxSegmentSizeBytes` → 500 MiB, `Payload` → default, `Headers` → idempotency + traceparent.

---

## 4. Connection & channel management — classic sink

### 4.1 Cache key

```go
type clientKey struct {
    url string // rendered amqp(s):// URL string
}
```

Since auth lives in the URL, the URL alone keys the connection. A `map[clientKey]*rabbitConn` guarded by `sync.Mutex` lives at package scope (mirrors `kafka/client.go` and `nats/client.go`).

### 4.2 `rabbitConn`

```go
type rabbitConn struct {
    url string

    mu         sync.Mutex            // serializes Publish+Confirm+Return-drain
    conn       *amqp.Connection      // current connection; nil while reconnecting
    ch         *amqp.Channel         // single channel in confirm mode
    confirms   chan amqp.Confirmation
    returns    chan amqp.Return
    closeNotif chan *amqp.Error

    reconciler *exchangeReconciler   // per (connection, exchange) declaration guard
    ready      chan struct{}         // closed when conn/ch are usable; replaced on reconnect
}
```

### 4.3 Connect

```go
amqp.DialConfig(url, amqp.Config{
    Heartbeat: 10 * time.Second,
    Dial:      amqp.DefaultDial(10 * time.Second),
    Properties: amqp.Table{
        "connection_name": "hump",
        "product":         "hump",
        "version":         version.Version,
    },
})
```

After dial:

1. `conn.Channel()` once, store as `ch`.
2. `ch.Confirm(false)` — enable publisher confirms.
3. `confirms = ch.NotifyPublish(make(chan amqp.Confirmation, 1))`.
4. `returns  = ch.NotifyReturn(make(chan amqp.Return, 1))`.
5. `closeNotif = conn.NotifyClose(make(chan *amqp.Error, 1))`.
6. Spawn supervisor goroutine (see §4.4).
7. Register `shutdown.OnShutdown("rabbitmq_sink(<url>)", ...)` that closes `ch` then `conn`.
8. Close `ready`.

### 4.4 Reconnect loop (background supervisor)

Per-connection goroutine:

```go
for {
    err, ok := <-rc.closeNotif
    if !ok { return }    // closed via shutdown
    logger.Warn("RabbitMQ connection closed", "url", rc.url, "err", err)

    rc.mu.Lock()
    rc.ch = nil
    rc.conn = nil
    rc.ready = make(chan struct{})
    rc.mu.Unlock()

    // Reconnect with backoff matching NATS:
    //   wait = 2s + jitter([100ms, 1s])
    //   infinite attempts; abort if shutdown is closing.
    for {
        wait := 2*time.Second + jitter(100*time.Millisecond, time.Second)
        select {
        case <-shutdown.Done():   return
        case <-time.After(wait):
        }
        if err := rc.dialAndOpen(); err != nil {
            logger.Warn("RabbitMQ reconnect failed", "url", rc.url, "err", err)
            continue
        }
        logger.Info("RabbitMQ reconnected", "url", rc.url)
        rc.reconciler.invalidate()   // re-declare exchanges on next publish
        break
    }
}
```

`Process()` callers `select` on `rc.ready` (with `ctx.Done()`) before holding `rc.mu` to publish. Until ready, the call returns a transient error so the retry layer can wait and try again on a new attempt.

### 4.5 Channel model

**Single channel, mutex-serialized publishes.** Channels in amqp091 are not goroutine-safe. Concurrency (from the existing delivery layer's `maxConcurrent`) is bounded by this mutex in v1 — the simplest design that matches the user's choice. If a channel-level error closes the channel without dropping the connection, the supervisor re-opens it lazily inside `Process()` under `rc.mu` before publishing.

---

## 5. Exchange reconciler (classic sink)

`exchangeReconciler` mirrors `internal/sinks/nats/reconciler.go`:

```go
type exchangeState struct {
    mu   sync.Mutex
    done bool
    err  error
}

type exchangeReconciler struct {
    chFn   func() (*amqp.Channel, error) // returns the cached confirm channel
    mu     sync.Mutex
    states map[string]*exchangeState     // keyed by rendered exchange name
}
```

`ensureDeclared(ctx, exchangeCfg, renderedName)` flow:

1. Snapshot the per-exchange state. If `done` and `err == nil`, return success.
2. Take per-exchange mutex. If `done`, return cached `err`.
3. Run declare:
    - `passive` → `ch.ExchangeDeclarePassive(name, type, durable, autoDelete, false, false, args)`. Channel error 404 (NOT_FOUND) → permanent (`errExchangeMissing`).
    - `create-when-missing` → try `ExchangeDeclarePassive` first; on 404 fall through to `ExchangeDeclare` with the configured params. On 406 PRECONDITION_FAILED (existing exchange has different settings) → permanent (`errExchangeMismatch`).
4. Transient transport/channel-close errors → return without marking `done`. Next publish retries.
5. On reconnect, `invalidate()` resets the map so the first publish after reconnect re-declares.

`Process()` calls `reconciler.ensureDeclared(...)` before `buildMessage`.

---

## 6. Publish flow — classic sink

```go
func Process(ctx context.Context, cfg config.RabbitMQSink, opCtx *sink.SinkContext) error {
    if err := ctx.Err(); err != nil { return err }

    rc, err := getClient(cfg, opCtx)
    if err != nil { return wrapError(err) }

    select {
    case <-rc.ready:
    case <-ctx.Done():
        return ctx.Err()
    }

    msg, err := buildMessage(cfg, opCtx)        // includes rendered exchange & routing key
    if err != nil { return wrapError(err) }

    if err := rc.reconciler.ensureDeclared(ctx, cfg.Exchange, msg.Exchange); err != nil {
        return wrapError(err)
    }

    rc.mu.Lock()
    defer rc.mu.Unlock()

    // Drain any stale confirmation/return left over from prior publishes (defensive).
    drainPending(rc.confirms, rc.returns)

    if err := rc.ch.PublishWithContext(
        ctx,
        msg.Exchange,
        msg.RoutingKey,
        msg.Mandatory,          // from cfg
        false,                  // immediate — always false (AMQP deprecated)
        msg.Publishing,         // amqp.Publishing built by buildMessage
    ); err != nil {
        return wrapError(err)
    }

    return awaitPublishResult(ctx, rc, msg)
}
```

`awaitPublishResult`:

1. Wait for either the next `amqp.Confirmation` on `rc.confirms`, `ctx.Done()`, or a Return on `rc.returns`.
2. RabbitMQ guarantees ordering: when `mandatory=true` and a message is unroutable, the broker sends `basic.return` **before** the corresponding `basic.ack`. So:
    - If a `Return` arrives first, drain the following `Confirmation` (which will be an ack — the broker still acks unroutable mandatory messages after returning them) and return `errUnroutable` (permanent).
    - If a `Confirmation` arrives with `Ack=true`, run a short non-blocking drain on `rc.returns` (50 ms timeout) to catch any straggler Return for this publish. Empty → success.
    - If a `Confirmation` arrives with `Ack=false` (NACK) → return `errPublishNacked` (permanent).
    - If `ctx.Done()` fires first → return `ctx.Err()` (transient via §8).

The 50 ms straggler window is defensible because the mutex guarantees no overlapping publishes: a Return that arrives after the ack must belong to this publish.

### `buildMessage`

Renders `exchange.name`, `routingKey`, `payload`, every `properties.*` template, every `headers[*]` template. Returns a struct:

```go
type rabbitMessage struct {
    Exchange   string
    RoutingKey string
    Mandatory  bool
    Publishing amqp.Publishing
}
```

`Publishing.DeliveryMode = amqp.Persistent` if `cfg.DeliveryMode == "persistent"` else `amqp.Transient`. `Publishing.Priority` parsed from rendered string to `uint8` (range check, render error if invalid). `Publishing.Headers = amqp.Table` built from rendered `headers` map, skipping empty values (mirrors Kafka/NATS).

Cross-sink default templates `payload.tmpl`, `idempotency_header.tmpl`, `traceparent_header.tmpl` are reused.

---

## 7. Connection & publish flow — stream sink

### 7.1 Cache

```go
type envKey struct { url string }
type producerKey struct {
    url    string
    stream string  // rendered stream name
}

var (
    envsMu      sync.Mutex
    envs        = map[envKey]*stream.Environment{}

    producersMu sync.Mutex
    producers   = map[producerKey]*stream.Producer{}
)
```

`stream.Environment` is created via `stream.NewEnvironment(stream.NewEnvironmentOptions().SetUri(url))`. One environment per URL, cached. `shutdown.OnShutdown` closes the environment.

`stream.Producer` is created per rendered stream name. Cached. `shutdown.OnShutdown` closes each producer. Producer creation is wrapped with the configured stream-reconcile call so the stream exists first.

### 7.2 Reconnect

The Go stream client exposes `SetReconnectStrategy` / `IsClosed` and emits a `ProducerClose` channel. The sink registers a callback: when a producer closes unexpectedly, evict it from `producers`; the next publish recreates it. The supervisor goroutine for the environment logs disconnect/reconnect transitions via the environment's `MetadataListener`. Backoff matches the classic sink (2 s + jitter, infinite).

### 7.3 Stream reconciler

Per (URL, stream-name) state. On first publish:

- `passive` → `env.StreamExists(name)`; if absent → permanent (`errStreamMissing`).
- `create-when-missing` → `env.DeclareStream(name, stream.NewStreamOptions().SetMaxAge(...).SetMaxLengthBytes(...).SetMaxSegmentSizeBytes(...))`. Already-exists error is treated as success; precondition mismatch (different retention) → permanent.

Transient errors are not cached; next publish retries.

### 7.4 Publish

```go
func Process(ctx context.Context, cfg config.RabbitMQStreamSink, opCtx *sink.SinkContext) error {
    if err := ctx.Err(); err != nil { return err }

    streamName, payload, headers, err := renderAll(cfg, opCtx)
    if err != nil { return wrapError(err) }

    prod, err := getProducer(ctx, cfg, opCtx, streamName)
    if err != nil { return wrapError(err) }

    msg := amqp.NewMessage([]byte(payload))
    msg.ApplicationProperties = toAMQPProps(headers)

    confirmCh := prod.NotifyPublishConfirmation()
    if err := prod.Send(msg); err != nil { return wrapError(err) }

    select {
    case conf := <-confirmCh:
        if !conf[0].IsConfirmed() { return wrapError(errPublishNacked) }
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

`prod` is shared across goroutines; the stream client's `Send` is safe to call concurrently. `NotifyPublishConfirmation` returns a shared channel for the producer — synchronous wait per publish is acceptable in v1; if throughput needs grow we can switch to a per-publish correlation map. Documented as a v1 simplification.

---

## 8. Error classification

`errors.go` in each package mirrors NATS:

```go
func wrapError(err error) error {
    if errors.Is(err, template.ErrRender) || isPermanentRabbitError(err) {
        return fmt.Errorf("rabbitmq: %w", errors.Join(err, errs.ErrSinkPermanent))
    }
    return fmt.Errorf("rabbitmq: %w", err)
}
```

**Classic sink — permanent:**

- `template.ErrRender` (shared).
- `errExchangeMissing` (passive declare 404).
- `errExchangeMismatch` (declare 406 PRECONDITION_FAILED).
- `errUnroutable` (mandatory + Return received).
- `errPublishNacked` (broker NACK on confirm).
- `amqp.Error` with `Code = 403 (ACCESS_REFUSED)`, `530 (NOT_ALLOWED)`, `404 (NOT_FOUND)`, `406 (PRECONDITION_FAILED)`, `541 (INTERNAL_ERROR)` only when received as a channel-level error (vs connection-level — those clear on reconnect).

**Classic sink — transient (default):**

- `amqp.ErrClosed` (connection/channel closed mid-publish — supervisor reconnects).
- `amqp.Error` with `Code = 320 (CONNECTION_FORCED)`, `501 (FRAME_ERROR)`, `504 (CHANNEL_ERROR)`, `505 (UNEXPECTED_FRAME)`.
- Confirm timeout via `context.DeadlineExceeded` / `context.Canceled`.
- Any net error during dial.

**Stream sink — permanent:**

- `template.ErrRender`.
- `errStreamMissing` (passive declare on absent stream).
- `errStreamMismatch` (declare with conflicting params).
- `errPublishNacked` (publish confirmation reports failure).
- Stream client errors: `stream.AuthenticationFailure`, `stream.AccessRefused`, `stream.PreconditionFailed`.

**Stream sink — transient (default):**

- `stream.StreamDoesNotExist` only when reconcile is `create-when-missing` and concurrent deletion is observed (retried).
- Connection/socket errors, deadline/cancel.

---

## 9. Cross-validation

`(*RabbitMQSink).CrossValidate(tables []Tables) error` mirrors `NatsJetStreamSink.CrossValidate`:

- Key: `(rendered URL source, rendered exchange.name source)`.
- Compared values: `exchange.type`, `exchange.durable`, `exchange.autoDelete`, sorted `exchange.arguments` keys+values.
- Divergence returns `fmt.Errorf("%w: exchange %q: divergent type/durable/autoDelete/arguments", errs.ErrCrossValidate, name)`.

`(*RabbitMQStreamSink).CrossValidate`:

- Key: `(URL source, stream.name source)`.
- Compared values: `stream.maxAge`, `stream.maxLengthBytes`, `stream.maxSegmentSizeBytes`.
- Same error pattern.

Both registered through the existing `CrossValidate` dispatch.

---

## 10. Logging

Per the chosen layout, two scopes:

- `internal/sinks/rabbitmq/logger.go` → `log.Scope("rabbitmq")`.
- `internal/sinks/rabbitmq_stream/logger.go` → `log.Scope("rabbitmq-stream")`.

Log lines emitted at lifecycle transitions only (mirrors Kafka's `connHooks` discipline — no per-publish logging):

| Event | Level | Fields |
|-------|-------|--------|
| Connection established | Info | url |
| Connection closed | Warn | url, err |
| Connection reconnected | Info | url |
| Reconnect attempt failed | Warn | url, err |
| Exchange / stream declared | Info | url, exchange/stream, mode |
| Channel reopened after channel-level error | Info | url, prev_err |

---

## 11. Defaults — quick reference

| Field | Default |
|-------|---------|
| `rabbitmq.exchange.name` | `hump.{{.Schema}}` |
| `rabbitmq.exchange.type` | `topic` |
| `rabbitmq.exchange.durable` | `true` |
| `rabbitmq.exchange.autoDelete` | `false` |
| `rabbitmq.exchange.mode` | `create-when-missing` |
| `rabbitmq.routingKey` | `{{.Schema}}.{{.Table}}` |
| `rabbitmq.mandatory` | `true` |
| `rabbitmq.deliveryMode` | `persistent` |
| `rabbitmq.payload` | `payload.tmpl` |
| `rabbitmq.properties.messageId` | `idempotency_header.tmpl` |
| `rabbitmq.properties.contentType` | `application/json` |
| `rabbitmq.headers` | `{X-Idempotency-Key, traceparent}` if absent |
| `rabbitmq-stream.stream.name` | `hump.{{.Schema}}.{{.Table}}` |
| `rabbitmq-stream.stream.mode` | `create-when-missing` |
| `rabbitmq-stream.stream.maxAge` | `24h` |
| `rabbitmq-stream.stream.maxLengthBytes` | `2 GiB` (2147483648) |
| `rabbitmq-stream.stream.maxSegmentSizeBytes` | `500 MiB` (524288000) |
| `rabbitmq-stream.payload` | `payload.tmpl` |
| `rabbitmq-stream.headers` | `{X-Idempotency-Key, traceparent}` if absent |
| Heartbeat (both sinks) | `10s` |
| Dial timeout (both sinks) | `10s` |
| Reconnect base wait | `2s` |
| Reconnect jitter | `[100ms, 1s]` |
| Confirm/return wait | bounded by `ctx` from retry's `attemptTimeout` |
| Return straggler drain (classic) | `50 ms` |

---

## 12. Dispatch wiring

`internal/types/config/config_sink_base.go` `unmarshalSink` switch gains:

```go
case sink.SinkTypeRabbitMQ:
    var r RabbitMQSink
    if err := yaml.NodeToValue(value, &r, yaml.Strict()); err != nil { return nil, err }
    return &r, nil

case sink.SinkTypeRabbitMQStream:
    var r RabbitMQStreamSink
    if err := yaml.NodeToValue(value, &r, yaml.Strict()); err != nil { return nil, err }
    return &r, nil
```

The framework's `Process` dispatcher (wherever sinks are routed by `Type()`) gains the two new cases calling `rabbitmq.Process` and `rabbitmq_stream.Process`.

---

## 13. Dev environment

### 13.1 `dev/docker-compose.yml`

Add a `rabbitmq` service using `rabbitmq:4-management` with the stream plugin enabled via an entrypoint that runs `rabbitmq-plugins enable rabbitmq_stream`. Expose ports `5672` (AMQP), `15672` (management UI), `5552` (stream).

```yaml
rabbitmq:
    image: rabbitmq:4-management
    restart: unless-stopped
    hostname: rabbitmq
    ports:
        - "5672:5672"
        - "15672:15672"
        - "5552:5552"
    environment:
        RABBITMQ_DEFAULT_USER: hump
        RABBITMQ_DEFAULT_PASS: hump
        RABBITMQ_SERVER_ADDITIONAL_ERL_ARGS: "-rabbitmq_stream advertised_host localhost"
    volumes:
        - rabbitmq-data:/var/lib/rabbitmq
        - ./initdb/rabbitmq-enabled-plugins:/etc/rabbitmq/enabled_plugins:ro
```

New file `dev/initdb/rabbitmq-enabled-plugins`:

```
[rabbitmq_management,rabbitmq_stream,rabbitmq_prometheus].
```

Add `rabbitmq-data:` to the `volumes:` block.

No init container is needed — exchange/stream declaration is done by hump on first publish.

### 13.2 `dev/hump.yaml`

Append two sinks to `sinks:` and add both to every `tables[].sinks` entry alongside the existing kafka/nats sinks (per the chosen scope: exercise all sinks together):

```yaml
sinks:
    # ... existing kafka1, nats-core, nats-jetstream ...
    - name: rabbitmq1
      rabbitmq:
          url: "amqp://hump:hump@localhost:5672/"
          exchange:
              name: "hump.{{.Schema}}"
              type: topic
          routingKey: "{{.Schema}}.{{.Table}}"
    - name: rabbitmq-stream
      rabbitmq-stream:
          url: "rabbitmq-stream://hump:hump@localhost:5552/%2F"
          stream:
              name: "hump.{{.Schema}}.{{.Table}}"

tables:
    - tables:
          - schema: public
            table: users
          - schema: public
            table: posts
          - schema: public
            table: comments
          - schema: public
            table: events
      sinks:
          - preset: kafka1
          - preset: nats-core
          - preset: nats-jetstream
          - preset: rabbitmq1
          - preset: rabbitmq-stream
```

---

## 14. Out of scope (v1)

- AMQP 1.0 protocol support.
- Channel pool / multi-channel concurrency for the classic sink (single-channel + mutex only).
- Super-streams / stream partitions / templated routing across multiple streams beyond the templated `stream.name`.
- Server-side stream filtering (`stream filter value`).
- Stream deduplication via `publishingId` (no broker-side dedup — retries rely on framework idempotency + downstream handling).
- Custom queue / binding declaration (sink owns exchange/stream only; queues are consumer-owned).
- Per-message TLS/cert auth (URL-only auth in v1; `amqps://` works but no client certs).
- Exposing `heartbeat` / `dialTimeout` / reconnect parameters as YAML fields.

## 15. Open implementation notes

- **`mandatory` is a `*bool`** in the YAML struct so that an unset field defaults to `true`. A plain `bool` would make `false` indistinguishable from "unset" given YAML semantics.
- **`Exchange.Arguments` map ordering** for cross-validation: marshal to canonical JSON for the comparison key (Go map iteration is unordered; comparing rendered keys requires sorting).
- **`returns.go` correlation invariant**: relies on the channel-level mutex serializing publishes. If we ever move to a channel pool, return correlation must move to a per-publish channel matched by an in-flight queue (`map[deliveryTag]chan ReturnOrAck`). Document this in the package comment so the future reader knows the constraint.
- **`Process` returning a transient error while `rc.ready` is not yet closed** must use a sentinel like `errReconnecting` so the retry layer (which uses `errors.Is(err, errs.ErrSinkPermanent)` for permanence) classifies it correctly — no `Join(errs.ErrSinkPermanent)`.
- **Dependency additions** in `go.mod`: `github.com/rabbitmq/amqp091-go`, `github.com/rabbitmq/rabbitmq-stream-go-client`. Run `go mod tidy` and verify no transitive license issues.
