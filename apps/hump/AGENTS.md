# apps/hump

The hump Go application (module `hump`). The project description, dependency policy and
pre-finish checks are in the root `AGENTS.md`. Paths below are relative to `apps/hump`.

Entry points: `cmd/hump` (the service), `cmd/openapi` (writes the OpenAPI spec for the web UI;
run through `moon run hump-tower:openapi`).

`internal/controller/hump_tower/` is the web UI build, produced by `moon run hump-tower:build`
and embedded into the binary. It is committed; never edit it by hand.

## Architecture

The Go app follows clean architecture: dependencies point inwards, towards `domain`.

| Directory | Role | May import |
|---|---|---|
| `internal/domain` | Domain definition: named types and constants, no behaviour, no I/O | nothing |
| `internal/port` | Ports: interfaces and the types they exchange, one package per capability | `domain` |
| `internal/adapter` | Adapters: port implementations over PostgreSQL, sinks, metrics etc. | `port`, `domain`, `config`, `pkg` |
| `internal/application` | Scoped use cases, grouped by scope (`processing`, `hump_tower`), one package per use case | `port`, `domain`, `config`, `pkg` |
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
  has neither, `processing/process_operation` and `hump_tower/stop_snapshot` have no `dto.go`).
  Transport types stay out of `usecase.go`.
- **A new external system** means a port in `internal/port` first, then its adapter, then the
  wiring in `cmd/hump`.

Packages under `internal/` carry their own `AGENTS.md` with the details. The sink packages
(`internal/adapter/sink/{http,kafka,nats,rabbitmq,rabbitmq_stream}`) have none; they are covered
by `internal/adapter/sink/AGENTS.md`.

## Linting

`moon run hump:lint` runs golangci-lint with `.golangci.yml`. Its `depguard` rules enforce the
layer table's "May import" column and the adapter-only-in-`cmd/hump` rule, but not "no sideways
imports": that one stays a review check. `moon run hump:lint-fix` applies the automatic fixes.
