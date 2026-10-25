# config — how this package hangs together

The table configuration: YAML text in, a validated `Configuration` with every default filled in
out. `Parse` is the only entry point and runs once, in `cmd/hump/main.go`, on the text
`pkg/app/options` resolved (sources and their order: `pkg/app/AGENTS.md`). Nothing is re-read or
reloaded afterwards. The user-facing schema is at https://hump.dev/docs/getting-started (outside
this repo), templating in `docs/template.md`; this file is about what the code does around it.

## Layout

| Path | Role |
|---|---|
| `parser.go` | `Parse`: expand → strict unmarshal with validation → cross-validate |
| `expand.go` | the YAML-to-YAML pre-pass: table groups, presets, `operations`, merge |
| `decoder/` | the shared `validator` instance and the strict `Decode` every type unmarshals through |
| `errors/` | `ErrCrossValidate`, the only sentinel |
| `resilience/` | `Failure`, `Retry`, `Backoff` — the per-sink policies |
| `sink/config_sinks.go` | `unmarshalSink` (the discriminator switch) and the `crossValidators` list |
| `sink/config_utils.go` | `getSinkType` and `nodeWithoutSinkTypes`: find the sink type key and strip it before decoding |
| `sink/config_nats.go`, `sink/config_rabbitmq.go` | `NatsCommon` / `RabbitMQCommon`, inlined into the two variants of each |

`decoder` and `errors` exist as packages only to break the cycle: `config` imports `config/sink`,
and both need the validator; `config/sink` returns the sentinel, `config` reaches it only in
tests.

## `Parse` is two passes over two different documents

1. `expand` builds an ordered `yaml.MapSlice` from the source's AST (`expand_value.go`), rewrites
   it, and **marshals it back to a string**. Strings are strings, null is nil, and every other
   scalar is a `sourceScalar` written back as typed, so `table: 00000` stays `00000`. A string goccy
   would not read back from its own output (`"\t"`, which it writes bare and reads as null) is
   written double-quoted. goccy's own decode still runs first, only to report syntax errors and
   duplicate keys.
2. That string, not the user's, is unmarshalled into `Configuration` with `decoder.Unmarshal`.
3. `crossValidate` compares sinks across tables.

So the typed structs never see `preset`, a nested `tables` group, a table-level `operations` or
the top-level `sinks` section; those keys exist in `expand.go` only, as string literals. And
every `[line:column]` and source snippet in an error points into the **expanded** text: comments
gone, presets inlined, `sinks` dropped, sequences re-indented. An unknown field inside a preset
is reported at the table that uses it, on a line the user's file does not have. The doc comment
on `Parse` does not say so.

Any other top-level key is carried through `expand` untouched and then rejected by strict mode.
`TestExpand_PreservesTopLevelOrderAndExtraKeys` covers only the first half; no test passes an
unknown top-level key through `Parse`.

## What `expand` does, in order

`flattenTables` → `resolveSinkPresets` → `pushDownOperations` → `mergeTablesByKey`. The order is
load-bearing: `operations` is pushed onto sinks **before** the merge because the merge keeps
only one entry's table keys (`TestParse_OperationsSurviveTableMerge`).

Every merge in this file is `mergeMapSlice(a, b)`: **top-level keys only**, `a` wins, `b`'s
extra keys are appended. That one rule explains all of the following.

- **Group → child.** A child of a `tables[].tables` group overrides the group's keys; the rest
  (`schema`, `replica`, `sinks`, `operations`) is inherited.
- **Preset override is shallow.** A sink entry's sibling keys replace the preset's keys
  wholesale: `preset: k1` next to `kafka: {topic: t2}` yields `kafka: {topic: t2}`. The
  preset's `url` is gone and the sink then fails `required`. The code does no deep merge.
- **Preset names are not validated.** `collectSinkPresets` keeps the **last** preset of a
  duplicated `name`, and a non-string `name` becomes the key `""`. Only an unknown reference is
  an error (`unknown sink preset: "x"`, a plain `fmt.Errorf` with no position).
- **Table `operations` replaces the sink's**, including one that came from a preset
  (`TestParse_TableOperationsOverrideSinkAndPreset`).
- **Duplicate tables.** Entries with the same `(schema, table)` collapse into the first. For a
  non-sink key the first entry's value wins *if it has one*; a key only a later entry sets
  (say `replica: unmanaged`) is adopted. Sinks are concatenated and deduplicated.

Two properties of the merge that are easy to get wrong:

- The key is the **literal** `schema` string. `table: x` and `schema: public, table: x` are two
  tables here, and nothing downstream maps `""` to `public` either. In `adapter/postgres`,
  `getTablesMetadata` (publication bootstrap, introspector) compares the schema literally, so an
  empty schema matches no table and ends in `ErrTablesNotFound`; only the `qualifiedTableName`
  paths let `search_path` resolve it. Traced in the code, not tested against a server.
- Sinks are deduplicated by their canonical YAML **before defaults are applied**. Two entries
  that differ only by a spelled-out default stay two sinks, and every row is delivered twice.

`expand` is lenient by construction: every type assertion that fails is skipped, a merge error
drops the entry, and a root that is not a mapping is returned unchanged. The typed pass is what
rejects malformed input, except a document with no mapping at all (empty, comments only), which
unmarshals into a zero `Configuration` with **no error**, because `validate:"min=1"` on `Tables`
never runs. `options` returns `ErrConfigMissing` only when no source sets either `Config` or
`ConfigPath`, so an empty `--config-file` gets through.

## Decoding: one pattern, repeated in every type

```go
func (t *T) UnmarshalYAML(node ast.Node) error {
	type alias T                                   // drops the method, avoids recursion
	if err := decoder.Decode(node, (*alias)(t)); err != nil {
		return err
	}
	t.setDefaults()
	return nil
}
```

`decoder.Decode` is `yaml.NodeToValue` with `yaml.Strict()` and the validator, which is what
turns a `validate:"…"` failure into a token-anchored `yaml.Error`. Two more things happen there:

- A `string`, `TableSchema` or `TableName` field gets a scalar's text as written (`scalar.go`):
  goccy would format `00000` back as `0` and `.inf` as `+Inf`.
- An unknown field is reported as the first unknown key in document order (`unknown_field.go`).
  goccy picks one of a mapping's unknown keys at random, and reports a nested one before those of
  the mapping around it. The walk stops at types with their own `UnmarshalYAML`, whose own
  `Decode` does the same.

A nested struct **without** its own `UnmarshalYAML` (`RabbitMQExchange`, `RabbitMQProperties`, `RabbitMQStreamConfig`,
`NatsStream`) is decoded and validated by its parent, and gets its defaults in the parent's
`setDefaults` (`NatsStream` gets none: `stream` is required). The inline-embedded `NatsCommon`
and `RabbitMQCommon` have no `UnmarshalYAML` either; the parent's `UnmarshalYAML` calls their
`setDefaults` / `SetDefaults` before its own.

Defaults are applied **after** validation, so `omitempty` on every optional field is what lets
an absent value through; the default itself is never validated. `Config.setDefaults` allocates
`Failure` and `Retry` when the keys are absent (that is why their `SetDefaults` are exported),
so after `Parse` neither pointer is nil and `process_operation` dereferences them unguarded.

`decoder.Validate` is one process-global instance. `config/sink` registers `natsAuthValidator`
on it from `init()`, so the "one NATS auth group" rule exists only in a binary that links that
package.

## A sink entry is a discriminated union

`sink.Config.UnmarshalYAML` works on the AST, not on a struct:

1. `getSinkType` looks for the keys in `port/sink.SinkTypes`. Zero **or more than one** gives
   `sink: must define exactly one of […]`.
2. `unmarshalSink` decodes that key's value into the concrete type.
3. `nodeWithoutSinkTypes` clones the mapping without the discriminator, and the rest (`failure`,
   `retry`, `operations`) is decoded strictly into `Config`, whose `Sink` field is `yaml:"-"`.

`MarshalYAML` writes the definition back under `Sink.Type()`. Only the debug dump in `main.go`
uses it, and the output contains every default, so it is not the user's config (`get_config`
returns `opts.Config` verbatim for that).

### The config structs are the port's `Definition`

`*KafkaSink`, `*HTTPSink` and the rest implement `port/sink.Definition`, and `adapter/sink/sink.go`
type-switches on these exact pointer types and passes the struct by value into the sink
subpackage. There is no mapping layer: a field added here is immediately what the adapter reads,
and a field renamed here is an adapter change. `DisplayName` and `Metadata` must keep their
`s == nil` guards: tests call them on typed-nil receivers (`TestSinkTypeAndDisplayName`,
`TestSinkMetadataNilReceiver`).

`Metadata(ctx)` returns `(host, target)` and discards render errors. Five callers must produce
identical strings from it: `topology()` in `main.go`, `process_operation`, `get_config`,
`get_snapshot_tables` and `adapter/snapshot`; see
`internal/application/hump_tower/get_config/AGENTS.md`.

### Adding a sink type

Nothing derives one list from another; skipping a step compiles.

1. `port/sink/sink_type.go`: the constant **and** `SinkTypes`. The latter is what makes the key
   a discriminator, and what `nodeWithoutSinkTypes` strips.
2. `sink/config_<type>.go`: struct, `UnmarshalYAML` + `setDefaults` (a type with no defaults
   of its own, like `NatsCoreSink`, calls only the common struct's), `Type`, `DisplayName`,
   `Metadata`, `CrossValidate`.
3. `sink/config_sinks.go`: a `case` in `unmarshalSink` (missing → falls to the "exactly one of"
   error) and an entry in `crossValidators`.
4. `adapter/sink/sink.go`: a `case` in the type switch (missing → `ErrUnknownSinkType` at
   delivery time, not at startup).
5. `README.md`.

## Cross-validation

Sinks that declare a broker object (a JetStream stream, an AMQP exchange, a RabbitMQ stream)
must agree on its settings wherever they name the same one, because the first publish reconciles
it. `CrossValidate` receives every sink of every table, grouped per table, and returns the first
divergence wrapped in `ErrCrossValidate`.

- `crossValidators` holds **typed-nil** pointers: `CrossValidate` is called on a nil receiver
  and must not touch `s`.
- Identity is `(URL source, object name)` joined with `\x00`, compared as **unrendered template
  source**. Two templates that render to the same exchange are not compared; one template that
  renders to different exchanges per table is compared as if it were one. JetStream's
  `stream.name` is a plain string, so there the check is exact.
- Values are compared **after defaults**, so an explicit default and an omitted one agree.
  The exception is JetStream `retention` and `storage`: `NatsJetStreamSink.setDefaults` does not
  fill them, so they are compared raw and `retention: limits` next to an omitted `retention`
  fails as divergent.
- JetStream compares `subjects` (order-insensitive), `retention`, `storage`. RabbitMQ compares
  exchange `type`, `durable`, `autoDelete`, `arguments` (as sorted JSON). RabbitMQ Streams
  compares `maxAge`, `maxLengthBytes`, `maxSegmentSizeBytes`. `mode` is not compared in any of them.
- No production code matches `ErrCrossValidate` with `errors.Is`; it only prefixes the startup
  error line.

## Default templates

Embedded from `sink/templ/`; a default is applied through `template.New` with the error dropped.

| Template | Used for |
|---|---|
| `payload.tmpl` | HTTP `body`; `payload` of every other sink (a Debezium-shaped envelope) |
| `idempotency_header.tmpl` | the `X-Idempotency-Key` header everywhere, RabbitMQ `properties.messageId`, JetStream `msgId` |
| `traceparent_header.tmpl` | the `traceparent` header everywhere |
| `client_id.tmpl` | HTTP `X-Client-ID` header, Kafka `clientId` |
| `kafka_topic.tmpl`, `nats_subject.tmpl`, `rabbitmq_routing_key.tmpl` | `{{ .Schema }}.{{ .Table }}` |
| `rabbitmq_exchange.tmpl` | `hump.{{ .Schema }}` |
| `rabbitmq_stream_name.tmpl` | `hump.{{ .Schema }}.{{ .Table }}` |

The files are embedded byte for byte: a trailing newline in a `.tmpl` becomes part of every
topic name or header value.

## `resilience`

- `Backoff` mirrors `pkg/resilience.Backoff` field for field (names, types, order), so
  `Policy()` is a plain type conversion. Adding a field to one and not the other is a compile
  error in `Policy()`, which is the point; keep them in step rather than writing a copy.
- `Retry.SetDefaults` fills the nested `Backoff` too, so `retry: {maxAttempts: 3}` and no
  `retry` at all end up with the same backoff.
- `Failure` accepts a scalar (`failure: skip`) or a mapping (`failure: {mode: skip}`). The scalar
  branch validates with `Validate.Struct` directly, so its error is a raw validator message
  without a position.
- `Increment` applies to linear only and `Multiplier` to exponential only, but nothing rejects
  the wrong combination, and `Multiplier` gets its default under `type: linear` as well.
  `Jitter` is commented "exponential only" but applies to both curves; it defaults to `full`
  (see `pkg/resilience/AGENTS.md`).

## Things that will bite

- **Headers are all-or-nothing.** Every sink applies its default headers only when `headers` is
  empty. One custom header removes `X-Idempotency-Key` and `traceparent` (and `Content-Type` /
  `X-Client-ID` for HTTP). Nothing pins this in a test.
- **A broken template passes `Parse`.** `{{ .x ` or a misspelled function is kept as static text
  and sent as is on every row; see `pkg/template/AGENTS.md`. A templated field also accepts
  only a string scalar: `tls: true` is a bool, but `priority: 5` under `properties` is an error.
- **An empty sink entry is an error.** `sinks: [~]` never reaches `sink.Config.UnmarshalYAML`,
  so `Table.checkSinks` rejects it (`sink entry 1 of table public.users is empty`). Without that
  it would leave a nil `Sink`, `Failure` and `Retry`, which `process_operation` and `main.go`
  dereference. A null entry under `tables` is dropped by `expand` instead.
- **Struct-level validator errors carry no position.** The NATS auth conflict surfaces as
  `Key: 'NatsCommon.auth' Error:Field validation for 'auth' failed on the 'natsauth' tag`.
- **Allowed values are repeated by hand.** `oneof=insert update delete truncate` (`sink/config.go`)
  and `oneof=full unmanaged` (`config.go`) restate `domain` constants; `oneof=linear exponential`,
  `full none`, `crash skip` restate `pkg/resilience` ones. See `internal/domain/AGENTS.md`.
- **`operations` defaults to a clone** of `domain.OperationTypes`; keep the `slices.Clone`, the
  slice is shared package state otherwise.
- `Configuration.ResolveAll()` just returns `Tables`. It exists because `process_operation`
  depends on a one-method interface rather than on this struct.

## Tests

- `parser_test.go`, `config_test.go`, `operations_test.go` drive almost everything through
  `Parse` with inline YAML, the behaviour a user sees. A new key or default gets a case there.
- `expand_test.go` tests each pass on `yaml.MapSlice` values and compares the expanded YAML as
  text; `TestCanonicalYAML_StableForEquivalentMaps` is what the sink dedup relies on.
- `sink/` tests are split per concern across all sink types, not per type.
- Not covered anywhere: the shallow preset override, duplicate preset names, `""` vs `public`,
  the all-or-nothing headers, a mapping-less document. A later duplicate table entry filling an
  absent key is covered only at unit level (`TestMergeTableEntries_NoSinksOnEither`), not
  through `Parse`.
