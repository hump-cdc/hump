# port/sink — how this package hangs together

The delivery end of the pipeline as the rest of the process sees it, and the most widely
imported port: `config/sink` implements `Definition` here, `adapter/sink` implements `Sink`,
and `process_operation`, `get_config`, `get_snapshot_tables`, `adapter/snapshot` and
`cmd/hump/main.go` consume them. It imports `domain` and the standard library, nothing else —
in particular not `config`, which depends on this package.

## Layout

| File | Role |
|---|---|
| `sink.go` | `Sink` (the adapter side) and `Definition` (the config side) |
| `sink_context.go` | `SinkContext` — one row change, and the root of every template |
| `sink_type.go` | `SinkType`, its constants, `SinkTypes` |
| `capture.go` | `Capture` — what an adapter actually sent, for the operations log |
| `errors.go` | `ErrUnknownSinkType`, `ErrSinkPermanent` |

## Two interfaces facing opposite ways

`Definition` is implemented by the YAML config structs in `config/sink/config_<name>.go`;
`Sink` is implemented once, by `adapter/sink/sink.go`, which type-switches on the concrete
`Definition` it is given and answers `ErrUnknownSinkType` for one it does not know. So
`Process` takes the definition as an argument on every call — there is one `Sink` for the
process, not one per configured sink.

The step-by-step contract of `Process` is in `adapter/sink/AGENTS.md`, along with the six
places adding a sink type touches.

## `SinkContext` is user-facing

Its exported fields are what a template sees as `.Version`, `.Timestamp`, `.Operation`,
`.Database`, `.Schema`, `.Table`, `.Headers`, `.Prev`, `.Next` — documented in README and
`docs/template.md`. Renaming or removing a field breaks users' config files. `Version` is the
build version (`version.Version`), set by `process_operation`.

It mirrors the parameters of `port/database.ProcessMessageHandler`; a field added to one
belongs in the other.

## `Definition.Metadata` is called with a partial context

`Metadata(SinkContext) (host, target)` produces the Prometheus labels and the topology keys.
Outside a delivery it is called with only `Schema` and `Table` set — by `process_operation`
`newBinding`, `cmd/hump/main.go` `topology`, `get_config` and `get_snapshot_tables` — and
`get_config` even passes the zero value to get a host. An implementation therefore must not
assume `Prev`, `Next`, `Headers` or `Operation` are populated, and must render the same strings
for the same `(Schema, Table)` everywhere: see `adapter/metrics_collector/AGENTS.md` for what
goes silently empty when they differ.

## `SinkType` is the YAML discriminator

The literals are the keys users write under a sink (`kafka:`, `nats-jetstream:` …), so a rename
is a breaking config change. `SinkTypes` is not decoration: `config/sink` `getSinkType` and
`nodeWithoutSinkTypes` iterate it to find and strip the discriminator, and the "must define
exactly one of" error prints it. A constant missing from the list is a sink type that cannot
be configured.

## `Capture`

Nil unless a client is watching the operations log. Its buffers belong to the caller and are
reused across deliveries, so an adapter writes through `Set` and `SetPayload` and never assigns
`Payload` or `Metadata`. The rest — when to fill it, the `header.` prefix — is on the type's doc
comment and in `adapter/sink/AGENTS.md`.

## `ErrSinkPermanent` has no reader

Adapters join it onto errors they judge non-retriable; nothing matches it, and the retry loop
retries regardless. Details in `adapter/sink/AGENTS.md`. Do not build on it without first
adding the consumer.
