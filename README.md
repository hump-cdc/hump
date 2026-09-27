# hump

```
docker compose -f dev/docker-compose.yml down -v
docker compose -f dev/docker-compose.yml up -d
```

## Configuration

The config file is a YAML document with two top-level keys (any other key is rejected):

| Key      | Required | Description                                             |
|----------|----------|---------------------------------------------------------|
| `sinks`  | no       | Named sink presets that can be referenced from `tables` |
| `tables` | yes      | Sequence of table entries with their sinks              |

### Tables

Each entry in `tables` is a mapping in one of two shapes. Mixing both shapes in the same entry (`tables` together with `schema`/`table`) is rejected.

**Single-table form** — applies the listed sinks to exactly one table:

```yaml
tables:
  - schema: public
    table: users
    sinks:
      - kafka:
          connection: "localhost:9092"
          topic: users
```

**Multi-table form** — applies the same `sinks` block to every inner table:

```yaml
tables:
  - tables:
      - schema: public
        table: orders
      - schema: public
        table: products
    sinks:
      - kafka:
          connection: "localhost:9092"
          topic: all_changes
          partitionKey: "{{ .schema }}.{{ .table }}"
```

The multi-table form is fanned out into one canonical single-table entry per inner table, each carrying a deep copy of the shared `sinks` block.

**`replica`** — optional, `full` (default) or `unmanaged`. With `full`, hump issues `ALTER TABLE … REPLICA IDENTITY FULL` on the table and its declarative partitions, which is what makes the old row and unchanged TOAST values reach the stream in full. With `unmanaged`, hump touches the table's replica identity not at all.

```yaml
tables:
  - schema: public
    table: audit_log
    replica: unmanaged
    sinks:
      - kafka: { connection: "localhost:9092", topic: audit }
```

Under any replica identity other than FULL, Postgres omits data hump cannot reconstruct: an unchanged TOAST value on `UPDATE`, and every non-key column of the old row when the update changes a key column or the row is deleted. Those columns reach templates as **unavailable** — they render as an empty string, and only `{{ unavailable . }}` tells them apart from a genuine `''`. See [Templating](#templating).

**`operations`** — optional list of row operations delivered to the table's sinks: any of `insert`, `update`, `delete`, `truncate`. Defaults to all four. Operations not in the list are skipped for that sink: they are not delivered, counted or logged.

```yaml
tables:
  - schema: public
    table: events
    operations: [insert, delete]
    sinks:
      - preset: nats2-jetstream

  - tables:
      - schema: public
        table: posts
      - schema: public
        table: comments
    operations: [insert, update]
    sinks:
      - preset: slow-http
```

`operations` may also be set on a single sink or in a sink preset. A table-level list takes precedence and replaces the sink's own. Without one, the sink's list applies, and with neither, all four operations are delivered. Because the list belongs to each sink, a table listed in several entries keeps a separate filter per entry's sinks.

### Sinks (table level)

Each item in a table's `sinks` sequence is a mapping that contains:

- exactly one sink-type discriminator key — `kafka:`, `http:`, `nats-core:`, `nats-jetstream:`, `rabbitmq:`, or `rabbitmq-stream:` (a sink with zero or more than one is rejected); **or**
- a `preset: <name>` reference to a top-level preset, optionally accompanied by a discriminator block that is deep-merged onto the preset.

Plus, at the same level, any of the optional keys: `failure`, `retry`, `operations` (see [`operations`](#tables)).

```yaml
sinks:
  - kafka:
      url: "localhost:9092"
      topic: users
    failure: skip
    retry:
      maxAttempts: 5
      attemptTimeout: 10s
      backoff:
        initial: 1s
        max: 30s
        exponential:
          multiplier: 2
```

### Sink presets

Top-level `sinks` is a sequence of preset mappings. Each preset must have a non-empty scalar `name`; duplicate names are rejected. The remaining keys are the same as a table-level sink (one discriminator key, plus optional policy fields).

```yaml
sinks:
  - name: k1
    kafka:
      url: "localhost:9092"
      topic: default
      partitionKey: "{{ .Next.id }}"
    failure: skip

tables:
  - schema: public
    table: tags
    sinks:
      - preset: k1                  # use preset as-is
      - preset: k1                  # inherit preset, override individual fields
        kafka:
          topic: tags
```

Override semantics: the preset is deep-copied (without `name`), the table-level sink is deep-copied (without `preset`), then the two are deep-merged — table-level mappings recurse into preset mappings, scalar overrides win. The merged sink must still contain exactly one discriminator key.

Rules:
- `name` is only valid on top-level presets, `preset` is only valid on table-level sinks.
- Referencing an unknown preset name is rejected.

### Sink: kafka

| Field          | Type             | Notes                                              |
|----------------|------------------|----------------------------------------------------|
| `url`          | template string  | Required. Comma-separated broker seed list.        |
| `topic`        | template string  | Default `{{ .Table }}`.                            |
| `partitionKey` | template string  |                                                    |
| `payload`      | template string  | Default emits a Debezium-style envelope.           |
| `headers`      | map of templates | Default `{X-Idempotency-Key, traceparent}`.        |
| `tls`          | bool             |                                                    |
| `sasl`         | string           | `plain`, `scram-sha-256`, or `scram-sha-512`.      |
| `username`     | template string  |                                                    |
| `password`     | template string  |                                                    |
| `clientId`     | template string  |                                                    |

```yaml
- kafka:
    url: "broker1:9092,broker2:9092"
    topic: "{{ .Schema }}.{{ .Table }}"
    partitionKey: "{{ coalesce .Next.id .Prev.id }}"
    headers:
      X-Source: hump
    tls: true
    sasl: scram-sha-256
    username: "{{ env \"KAFKA_USER\" }}"
    password: "{{ env \"KAFKA_PASS\" }}"
    clientId: hump-1
```

### Sink: http

| Field     | Type             | Notes                                              |
|-----------|------------------|----------------------------------------------------|
| `url`     | template string  | Required.                                          |
| `method`  | template string  | Default `POST`.                                    |
| `body`    | template string  | Default emits a Debezium-style envelope.           |
| `headers` | map of templates | Default `{Content-Type: application/json, X-Client-ID, X-Idempotency-Key, traceparent}`. |

```yaml
- http:
    url: "http://localhost:8080/events"
    method: POST
    headers:
      Content-Type: application/json
    body: |
      {"id": {{ .Next.id }}, "schema": "{{ .Schema }}"}
```

### Sink: nats-core

Publishes to a Core NATS subject (fire-and-forget with a synchronous flush).

| Field         | Type             | Notes                                              |
|---------------|------------------|----------------------------------------------------|
| `url`         | template string  | Required.                                          |
| `name`        | template string  | Client name; defaults to the host name.            |
| `subject`     | template string  | Default `{{ .Schema }}.{{ .Table }}`.              |
| `payload`     | template string  | Default emits a Debezium-style envelope.           |
| `headers`     | map of templates | Default `{X-Idempotency-Key, traceparent}`.        |
| `username`    | template string  | User/password auth.                                |
| `password`    | template string  |                                                    |
| `token`       | template string  | Token auth.                                        |
| `credentials` | template string  | Inline `.creds` file contents (JWT + nkey seed).   |
| `nkeySeed`    | template string  | Raw nkey seed.                                     |

At most one auth method may be configured per sink.

```yaml
- nats-core:
    url: "nats://localhost:4222"
    username: "hump"
    password: "hump"
    subject: "hump.{{ .Schema }}.{{ .Table }}"
```

### Sink: nats-jetstream

Publishes to a JetStream-backed subject with publisher acks and (optionally) a stream reconciler on first use. Inherits every `nats-core` field above and adds:

| Field                | Type             | Notes                                                     |
|----------------------|------------------|-----------------------------------------------------------|
| `stream.name`        | string           | Required. JetStream stream the publish asserts.           |
| `stream.subjects`    | list of strings  | When set, the reconciler declares/updates the stream.     |
| `msgId`              | template string  | Sets `Nats-Msg-Id` for server-side de-dup. Default `idempotency_header.tmpl`. |
| `mode`               | enum             | `create-or-update` (default) or `create-when-missing`.    |
| `retention`          | enum             | `limits`, `interest`, `workqueue`.                        |
| `storage`            | enum             | `file`, `memory`.                                         |

The reconciler only runs when `stream.subjects` is non-empty.

```yaml
- nats-jetstream:
    url: "nats://localhost:4222"
    username: "hump"
    password: "hump"
    subject: "hump.js.{{ .Schema }}.{{ .Table }}"
    stream:
      name: HUMP
      subjects:
        - "hump.js.>"
    retention: limits
    storage: file
```

### Sink: rabbitmq

Publishes to a classic AMQP 0.9.1 exchange with publisher confirms and (when `mandatory: true`) `basic.return` correlation. The sink owns its single mutex-serialized channel per cached connection.

| Field                          | Type             | Notes                                                    |
|--------------------------------|------------------|----------------------------------------------------------|
| `url`                          | template string  | Required. `amqp://` or `amqps://`.                       |
| `exchange.name`                | template string  | Default `hump.{{ .Schema }}`.                            |
| `exchange.type`                | enum             | `direct`, `topic` (default), `fanout`, `headers`.        |
| `exchange.durable`             | bool             | Default `true`.                                          |
| `exchange.autoDelete`          | bool             | Default `false`.                                         |
| `exchange.mode`                | enum             | `passive` or `create-when-missing` (default).            |
| `exchange.arguments`           | map of any       | Optional `x-*` declaration args.                         |
| `routingKey`                   | template string  | Default `{{ .Schema }}.{{ .Table }}`.                    |
| `mandatory`                    | bool             | Default `true`. `false` silently drops unroutable msgs.  |
| `deliveryMode`                 | enum             | `persistent` (default) or `transient`.                   |
| `payload`                      | template string  | Default emits a Debezium-style envelope.                 |
| `properties.messageId`         | template string  | Default `idempotency_header.tmpl`.                       |
| `properties.correlationId`     | template string  |                                                          |
| `properties.type`              | template string  |                                                          |
| `properties.appId`             | template string  | Default `hump`.                                          |
| `properties.userId`            | template string  | Validated by the broker against the connection user.     |
| `properties.contentType`       | template string  | Default `application/json`.                              |
| `properties.contentEncoding`   | template string  |                                                          |
| `properties.expiration`        | template string  | Per-message TTL in ms (as string).                       |
| `properties.priority`          | template string  | Rendered string parsed to `0..255`.                      |
| `headers`                      | map of templates | Default `{X-Idempotency-Key, traceparent}`.              |

Cross-sink rule: every sink pointing at the same `(url, exchange.name)` source must agree on `type` / `durable` / `autoDelete` / `arguments`.

```yaml
- rabbitmq:
    url: "amqp://hump:hump@localhost:5672/"
    exchange:
      name: "hump.{{ .Schema }}"
      type: topic
    routingKey: "{{ .Schema }}.{{ .Table }}"
```

### Sink: rabbitmq-stream

Publishes to a RabbitMQ Streams stream with publisher confirmations.

| Field                         | Type             | Notes                                                     |
|-------------------------------|------------------|-----------------------------------------------------------|
| `url`                         | template string  | Required. `rabbitmq-stream://` (or `rabbitmq-stream+tls://`). |
| `stream.name`                 | template string  | Default `hump.{{ .Schema }}.{{ .Table }}`.                |
| `stream.mode`                 | enum             | `passive` or `create-when-missing` (default).             |
| `stream.maxAge`               | Go duration      | Default `24h`.                                            |
| `stream.maxLengthBytes`       | int64            | Default `2 GiB`.                                          |
| `stream.maxSegmentSizeBytes`  | int64            | Default `500 MiB`.                                        |
| `payload`                     | template string  | Default emits a Debezium-style envelope.                  |
| `headers`                     | map of templates | Rendered into AMQP 1.0 application-properties.            |

Cross-sink rule: every sink pointing at the same `(url, stream.name)` source must agree on `maxAge` / `maxLengthBytes` / `maxSegmentSizeBytes`.

```yaml
- rabbitmq-stream:
    url: "rabbitmq-stream://hump:hump@localhost:5552/%2F"
    stream:
      name: "hump.{{ .Schema }}.{{ .Table }}"
      maxAge: 168h
```

### Failure policy

Scalar enum at the sink level:

```yaml
failure: skip
```

```yaml
failure: crash
```

| Value   | Meaning                                       |
|---------|-----------------------------------------------|
| `crash` | Fail the process when delivery fails.         |
| `skip`  | Drop the event and continue.                  |

Any other value is rejected.

Every sink is delivered attached: the LSN advances only once the sink
completes, so the event a `crash` stops the process on is still recovered on
restart.

### Retry policy

`retry` is a mapping with `maxAttempts`, `attemptTimeout` and a `backoff` block. The `backoff` block must contain exactly one of `exponential:` or `linear:` (declaring both is rejected).

**Exponential backoff:**

```yaml
retry:
  maxAttempts: 5
  attemptTimeout: 10s
  backoff:
    initial: 1s
    max: 30s
    exponential:
      multiplier: 2
```

**Linear backoff:**

```yaml
retry:
  maxAttempts: 3
  attemptTimeout: 5s
  backoff:
    initial: 1s
    max: 10s
    linear:
      increment: 2s
```

| Field                            | Notes                                                                 |
|----------------------------------|-----------------------------------------------------------------------|
| `maxAttempts`                    | Integer                                                               |
| `attemptTimeout`                 | Go duration (e.g. `500ms`, `10s`, `1m`)                               |
| `backoff.initial`                | Go duration                                                           |
| `backoff.max`                    | Go duration                                                           |
| `backoff.exponential.multiplier` | Float; only valid under `exponential`                                 |
| `backoff.linear.increment`       | Go duration; only valid under `linear`                                |

### Templating

String fields marked "template string" are parsed as Go `text/template` sources. Header values are also templated. A static literal with no `{{ ... }}` actions is stored verbatim.

Every template renders against the per-event context:

| Field        | Type                  | Notes                                                            |
|--------------|-----------------------|------------------------------------------------------------------|
| `.Version`   | `string`              | Build version.                                                   |
| `.SinkType`  | `string`              | e.g. `kafka`, `rabbitmq-stream`.                                 |
| `.SinkName`  | `string`              | The sink's `DisplayName` (e.g. `kafka(localhost:9092)`).         |
| `.Timestamp` | `time.Time`           | WAL commit timestamp.                                            |
| `.Operation` | `string`              | `INSERT`, `UPDATE`, `DELETE`, `TRUNCATE`.                        |
| `.Database`  | `string`              | Source DB name.                                                  |
| `.Schema`    | `string`              |                                                                  |
| `.Table`     | `string`              |                                                                  |
| `.Headers`   | `map[string]string`   | Carries `traceparent` when present.                              |
| `.Prev`      | `map[string]any`      | Row before the change (`nil` on `INSERT`).                       |
| `.Next`      | `map[string]any`      | Row after the change (`nil` on `DELETE`).                        |

A column the source could not deliver — see [`replica`](#tables) — is **unavailable**: it renders as an empty string, compares equal to `""`, and counts as empty for `empty`, `coalesce` and `required`. `{{ unavailable .Next.blob }}` is the only way to distinguish it from a column that really is `''`:

```yaml
payload: |
  {{ if unavailable .Next.blob }}{"blob":"unchanged"}{{ else }}{{ .Next | json false }}{{ end }}
```

Standard `text/template` actions plus the bundled funcs (`json`, `sha256`, `cat`, `coalesce`, `unixepoch`, `mul`, `lower`, `trunc`, `replace`, `now`, `env`, `index`, …) are available. Example: `{{ coalesce .Next.id .Prev.id }}` extracts the row's id across INSERT/UPDATE/DELETE.

### Full example

```yaml
sinks:
  - name: k1
    kafka:
      url: "localhost:9092"
      topic: default
      partitionKey: "{{ coalesce .Next.id .Prev.id }}"
    retry:
      maxAttempts: 5
      attemptTimeout: 10s
      backoff:
        initial: 1s
        max: 30s
        exponential:
          multiplier: 2
  - name: h1
    http:
      url: "http://localhost:8080/users"
      method: POST
      body: |
        {"op": "{{ .Operation }}", "id": {{ .Next.id }}}
    failure: skip
  - name: nj1
    nats-jetstream:
      url: "nats://localhost:4222"
      username: "hump"
      password: "hump"
      subject: "hump.js.{{ .Schema }}.{{ .Table }}"
      stream:
        name: HUMP
        subjects: ["hump.js.>"]
      retention: limits
      storage: file
  - name: r1
    rabbitmq:
      url: "amqp://hump:hump@localhost:5672/"
      exchange:
        name: "hump.{{ .Schema }}"
        type: topic
      routingKey: "{{ .Schema }}.{{ .Table }}"
  - name: rs1
    rabbitmq-stream:
      url: "rabbitmq-stream://hump:hump@localhost:5552/%2F"
      stream:
        name: "hump.{{ .Schema }}.{{ .Table }}"

tables:
  - schema: public
    table: tags
    sinks:
      - preset: h1
      - preset: k1
        kafka:
          topic: tags

  - schema: public
    table: users
    sinks:
      - preset: k1
      - preset: nj1
      - preset: r1
      - preset: rs1

  - tables:
      - schema: public
        table: orders
      - schema: public
        table: products
    sinks:
      - kafka:
          url: "localhost:9092"
          topic: all_changes
          partitionKey: "{{ .Schema }}.{{ .Table }}"
```
