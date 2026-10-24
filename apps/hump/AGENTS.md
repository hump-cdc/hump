# apps/hump

The hump Go application (module `hump`). The project description, dependency policy and
pre-finish checks are in the root `AGENTS.md`. Paths below are relative to `apps/hump`.

Entry points: `cmd/hump` (the service), `cmd/openapi` (writes the OpenAPI spec for the web UI;
run through `moon run hump-tower:openapi`).

`internal/controller/hump_tower/` is the committed web UI build (`moon run hump-tower:build`);
never edit it by hand.

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
  imports `internal/adapter` (tests may, see below); `cmd/hump` builds the adapters and injects
  them as port interfaces.
- **No sideways imports.** A use case does not import another use case, a port does not
  import another port, an adapter does not import another adapter. The one exception is a
  registry over its own subpackages (`adapter/sink` → `adapter/sink/kafka` etc.).
- **Nothing imports `controller`** except `cmd/hump` and `cmd/openapi` (which builds it only to
  write the OpenAPI spec). Tests in `controller` also import `adapter`, `config` and `domain`.
- **A use case package is self-contained**: `usecase.go` holds the logic and its own result
  types, `controller.go` (if it has a transport) the handler that calls it, `dto.go` (if it has
  wire types) the wire types. Transport types stay out of `usecase.go`.
- **A new external system** means a port in `internal/port` first, then its adapter, then the
  wiring in `cmd/hump`.

Packages under `internal/` carry their own `AGENTS.md` with the details. The sink packages
(`internal/adapter/sink/{http,kafka,nats,rabbitmq,rabbitmq_stream}`) have none; they are covered
by `internal/adapter/sink/AGENTS.md`.

## Shared rules

- **Nil web UI dependencies.** The web UI ports (`live_stream`, `metrics_collector`,
  `operations_collector`, `snapshot`) are wired in `cmd/hump` only when `WEB_PORT` is set, and
  `cmd/openapi` builds every `hump_tower` use case with nil dependencies just to register routes.
  So the `hump_tower` use cases and `processing/process_operation` nil-check them. The one
  exception, `get_config`, is documented in its own `AGENTS.md`.
- **`dto.go` is the API contract.** Its types and struct tags end up in
  `apps/hump-tower/src/shared/api/openapi.json`; after changing one, run
  `moon run hump-tower:openapi`. See `internal/controller/AGENTS.md`.
- **Test helpers go below the last test function** in a `_test.go` file.
- **Import aliases.** When a package name differs from its directory (`idle_gate` →
  `idlegate`, `port/live_stream` → `livestream`, every use case package except
  `process_operation`), every import of it has an explicit alias.

## Linting

The `depguard` rules in `.golangci.yml` enforce the layer table's "May import" column and the
adapter-only-in-`cmd/` rule, but not "no sideways imports": that one stays a review check.
