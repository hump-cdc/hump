# hump

Change data capture for PostgreSQL: hump reads logical replication (WAL) changes — insert,
update, delete, truncate — for the configured tables and delivers them to sinks: Kafka, HTTP,
NATS (core / JetStream), RabbitMQ, RabbitMQ Streams. Payloads are templated; delivery has
retry and failure policies. It can also snapshot existing table rows, exposes Prometheus
metrics, and ships a web UI. Configuration is described in `README.md`.

## Stack

- Go application (module `hump`), entry point `cmd/hump`.
- Web UI on React — `web/webui` (TypeScript, rsbuild, bun). Its build output goes to
  `internal/controller/webui` and is embedded into the binary.

## Architecture

The Go app follows clean architecture: dependencies point inwards, towards `domain`.

| Directory | Role | May import |
|---|---|---|
| `internal/domain` | Domain definition: named types and constants, no behaviour, no I/O | nothing |
| `internal/port` | Ports: interfaces and the types they exchange, one package per capability | `domain` |
| `internal/adapter` | Adapters: port implementations over PostgreSQL, sinks, metrics etc. | `port`, `domain`, `config`, `pkg` |
| `internal/application` | Scoped use cases, grouped by scope (`processing`, `webui`), one package per use case | `port`, `domain`, `config`, `pkg` |
| `internal/controller` | Controllers / entry points: the WAL listener and the HTTP servers | `application`, `port`, `pkg` |
| `cmd/hump` | Composition root | everything |

Outside the layers:

- `pkg/` — re-usable utilities across the app; imports nothing from `internal` or `config`.
- `config/` — YAML configuration; imports `port`, `domain`, `pkg`.

Principles:

- **Use cases depend on ports, never on adapters.** No production code outside `cmd/hump`
  imports `internal/adapter` (tests may, see below); `cmd/hump` builds the adapters and injects them as port interfaces.
- **No sideways imports.** A use case does not import another use case, a port does not
  import another port, an adapter does not import another adapter. The one exception is a
  registry over its own subpackages (`adapter/sink` → `adapter/sink/kafka` etc.).
- **Nothing imports `controller`** except `cmd/hump` and `cmd/openapi` (which builds it only to
  write the OpenAPI spec). Tests in `controller` also import `adapter`, `config` and `domain`.
- **A use case package is self-contained**: `usecase.go` holds the logic and its own result
  types, `controller.go` the handler that calls it, `dto.go` the wire types (`controller.go` where
  the use case has a transport, `dto.go` where it has wire types; `processing/cleanup_database`
  has neither, `processing/process_operation` and `webui/stop_snapshot` have no `dto.go`).
  Transport types stay out of `usecase.go`.
- **A new external system** means a port in `internal/port` first, then its adapter, then the
  wiring in `cmd/hump`.

Packages under `internal/` carry their own `AGENTS.md` with the details. The sink packages
(`internal/adapter/sink/{http,kafka,nats,rabbitmq,rabbitmq_stream}`) have none; they are covered
by `internal/adapter/sink/AGENTS.md`.

## Before finishing a task

Every time you finish a task, run both linters and fix everything they report before you call
the task done:

- Go: `task lint` (golangci-lint with `.golangci.yml`; its `depguard` rules enforce the layer
  import rules above). `task lint:fix` applies the automatic fixes.
- Web UI: `bun run check` in `web/webui` (Biome: lint, format, import order).
  `bun run check --write` applies the safe fixes.
