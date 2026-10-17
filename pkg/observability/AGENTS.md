# pkg/observability — how this package hangs together

Three independent packages and no code at this level: logging, health probes and the Prometheus
registry. `cmd/hump/main.go` builds the health and metric objects once and hands them to the
adapters. `log` is imported directly by 16 of the module's 51 packages: `cmd/hump`,
`internal/controller`, both `processing` use cases, nine of the eleven adapter packages (all but
`adapter/sink` and `adapter/sink/http`), `pkg/app`, `pkg/app/shutdown` and `pkg/middleware`.
`domain`, the ports, `config` and the `webui` use cases do not import it.

## Layout

| File | Role |
|---|---|
| `log/log.go` | the slog default handler (set in `init`), `Scope`, `WithScope`, `WithTraceparent` |
| `log/traceparent.go` | the traceparent parser behind `WithTraceparent` |
| `health/health.go` | `Health` — the health-go instance and its two checks |
| `health/health_check.go` | `HealthCheck` — one flag a component flips |
| `health/errors.go` | `ErrNotOperational` |
| `metric/metric.go` | `New` — the registry, its factory and its handler |

There are no tests under this directory.

## log

**Importing the package configures logging.** `init` replaces `slog.Default()` based on two
environment variables read at that moment: `LOG_FORMAT=json` selects the JSON handler, anything else
the coloured `tint` handler (colour off when stdout is not a terminal), and `LOG_VERBOSE=true`
lowers the level to Debug. Output goes to **stdout**. Setting either variable after the process has
started does nothing.

`Scope(area)` snapshots `slog.Default()` and the build's `version`/`commit` into a `*Log`. The
convention across the repository is a package-level `var logger = log.Scope("<area>")`, usually in a
`logger.go`; it is safe as a package variable because Go initialises an imported package before its
importer. `version` is on every line (it defaults to `development`); `commit` only when the build
set it.

`*Log` embeds `*slog.Logger`, so it is used as one. `WithScope` has no callers yet; its comment says
it overrides the scope, but it is a plain `With("area", …)`, and slog's built-in handlers do not
deduplicate keys — a logger that already has an `area` ends up with two.

`WithTraceparent` attaches `trace_id` and `span_id` when the value parses, and falls back to
attaching the raw value as `trace_id` when it does not — it never drops what it was given. Its
parser is **not** the one behind the `trace_id`/`span_id` template functions in
`pkg/template/funcs/traceparent.go`:

| | `log` | `template/funcs` |
|---|---|---|
| Hex case | either | lowercase only |
| Version `ff`, all-zero ids | accepted | rejected |
| How it splits | fixed offsets, 55 bytes, no allocation | `strings.Split` on `-` |

So a traceparent can be logged with a clean `trace_id` and still be passed through unparsed by a
template. Keep that in mind before "unifying" them: the log side runs per row and is written not to
allocate.

## health

`New` hardcodes the two checks the process has, `Postgres` and `WAL Sender`, as exported fields; a
new check is a field, a `newHealthCheck` and a `configure()` in `New`. This package names
application components on purpose — it is the one place the probe list lives.

- A check starts **not operational**, so the probes fail until the adapter calls `MarkHealthy`.
- `MarkUnhealthy(err)` makes the probe report `err`; with a nil error it reports
  `ErrNotOperational`.
- State is an `atomic.Pointer`, so marking is safe from any goroutine and costs one allocation.
- `Handler()` is a single handler. `controller/observability.go` mounts it on `/health/startup`,
  `/health/ready` and `/health/live` alike — the three paths always agree.

`adapter/postgres` does not import this package; it declares the two-method interface it needs and
`main.go` passes `health.Postgres` and `health.WalSender` in.

## metric

`New(namespace)` creates a **fresh registry**, not the Prometheus default, and returns three views
of it:

- a `promauto.Factory` that prefixes everything registered through it with `namespace_` — every
  application metric is created through this, which is where `hump_` comes from;
- the `/metrics` handler;
- the registry as a `prometheus.Gatherer`, which `adapter/metrics_collector` scrapes in-process.

The Go runtime and process collectors are registered on the raw registry, so they are **not**
prefixed: `go_memstats_alloc_bytes`, not `hump_go_…`. The metrics collector reads that family by its
unprefixed name.

Anything registered on `prometheus.DefaultRegisterer` instead of the factory is invisible to both
`/metrics` and the dashboard.
