# port/sink — how this package hangs together

The delivery end of the pipeline as the rest of the process sees it, and the most widely
imported port: `config/sink` implements `Definition` here, `adapter/sink` implements `Sink`.
It imports `domain` and the standard library, nothing else —
in particular not `config`, which depends on this package.

`Sink` is implemented once, by `adapter/sink/sink.go`, which type-switches on the concrete
`Definition` it is given, so there is one `Sink` for the process, not one per configured sink.
The step-by-step contract of `Process`, the `header.` prefix for captured headers, and the six
places adding a sink type touches are in `adapter/sink/AGENTS.md`.

## `Context` is user-facing

Its exported fields are what a template sees as `.Version`, `.Timestamp`, `.MessageId`,
`.SnapshotID`, `.Operation`, `.Database`, `.Schema`, `.Table`, `.Headers`, `.Prev`, `.Next` — documented in the repo-root
`docs/template.md`. Renaming or removing a field breaks users' config files. `Version` is the
build version (`version.Version`), set by `process_operation`.

`process_operation` copies `MessageMetadata.MessageID` to `MessageId` (exact casing) and
`MessageMetadata.SnapshotID` to `SnapshotID`. `SnapshotID` is empty for WAL changes and holds
the exported database snapshot id for snapshot rows. `Timestamp` is the transaction's
`Begin.CommitTime` for WAL, and `time.Now()` per row for snapshots.

## `Definition.Metadata` is called with a partial context

`Metadata(Context) (host, target)` produces the Prometheus labels and the topology keys.
Outside a delivery it is called with only `Schema` and `Table` set, and `get_config` even passes the zero value to get a host. An implementation
therefore must not assume `Prev`, `Next`, `Headers` or `Operation` are populated, and must
render the same strings for the same `(Schema, Table)` everywhere, byte for byte: see
`adapter/metrics_collector/AGENTS.md` for what goes silently empty when they differ.

## `Type` is the YAML discriminator

The literals are the keys users write under a sink (`kafka:`, `nats-jetstream:` …), so a rename
is a breaking config change. `SinkTypes` is not decoration: `config/sink` `getSinkType` and
`nodeWithoutSinkTypes` iterate it to find and strip the discriminator, and the "must define
exactly one of" error prints it. A constant missing from the list is a sink type that cannot
be configured.

## `ErrSinkPermanent` has no reader

Adapters join it onto errors they judge non-retriable; nothing matches it, and the retry loop
retries regardless. Details in `adapter/sink/AGENTS.md`. Do not build on it without first
adding the consumer.
