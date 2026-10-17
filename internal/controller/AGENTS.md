# controller — how this package hangs together

The process's entry points: everything `cmd/hump/main.go` hands to the runner lives here. There is
no logic of its own — each controller binds a transport (WAL listener, HTTP port) to handlers built
elsewhere. Request mapping, status codes and DTOs belong to the per-use-case `Controller` in
`internal/application/...`; this package only decides which path and which port they answer on.

## Layout

| File | Role |
|---|---|
| `process.go` | `ProcessController` — the WAL listener, `db.Listen(ctx, process)` |
| `observability.go` | `ObservabilityController` — health probes and the Prometheus scrape on `TELEMETRY_PORT` |
| `webui.go` | `WebUIController` — the web UI API and the embedded SPA on `WEB_PORT` |
| `webui/` | the built SPA, embedded through `//go:embed webui` — a build artifact, see below |

## Three controllers, two runner lanes

- `ProcessController.Listen` goes through `runner.Run`: a **worker**. On shutdown its context is
  cancelled first and it is drained (up to 30s) before any shutdown hook runs — cancel means "take
  no new work", not "abort".
- Both `Serve`s go through `runner.Serve`: **servers**. They outlive the workers and stop only when
  their own `shutdown.StageServers` hook calls `server.Shutdown`. The hook is registered *inside*
  `Serve`, so a server that never got that far has no hook — `runner.Wait` joins servers with a
  timeout for exactly that case. `http.ErrServerClosed` is the clean exit and is swallowed.

`WebUIController.Serve` passes the runner's context as the server's `BaseContext`, so every request
context dies at the start of shutdown. That is what ends the SSE handlers (`/api/live`,
`/api/snapshot`, `/api/snapshot/estimate`); without it `server.Shutdown` would hang on the open
streams until the `StageServers` hook timed out. `ObservabilityController.Serve` takes no context — it has nothing long-lived.

Both servers are optional: `main.go` starts each only when its port option is set, and neither in
`--cleanup` mode.

`ProcessController` hands `process_operation.Controller.Process` to the database as the WAL
handler. `main.go` gives the same function to the snapshot runner; that is safe only because a
snapshot pauses WAL delivery, so the two never call it at once.

## `Serve` and `Api` are split on purpose

`Api(router)` registers the routes and returns the `huma.API`; `Serve` adds middleware, the static
handler and the listener around it. `Api` has three callers: `Serve`, `webui_test.go`, and
`cmd/openapi/main.go`, which builds the controller with nil dependencies (handlers are registered,
never invoked) and writes the spec to `web/webui/src/shared/api/openapi.json`. `openapi-ts` turns
that committed file into the TypeScript client. So a route declaration here is also the client's
source:

- `Tags` picks the SDK class (`byTags`) and whatever follows the first `.` of `OperationID`
  becomes the method — renaming either renames the generated client method.
- `Summary` and `DefaultStatus` (202 for `snapshot.start`, 204 for `snapshot.stop`) land in the
  spec; the status is not set anywhere else.
- `OpenAPIPath`, `DocsPath` and `SchemasPath` are emptied and `CreateHooks` is nil: nothing serves
  the spec or `$schema` links at runtime. `task openapi` is the only producer.
- The spec's `info.version` is `version.Version`, set by ldflags.

**Adding a route**: a `huma.Register` (or `sse.Register` with the use case's `Events()`) in
`routes`, and a field plus a `NewWebUIController` parameter — which has to be passed in three
places: `cmd/hump/main.go`, `cmd/openapi/main.go` and `newTestRouter` in `webui_test.go`. An SSE
route also needs a `flattenSSESchema` call in `cmd/openapi/main.go` listing its event schemas;
the comment there explains why, and why the payloads must stay distinguishable by shape alone.
Then `task openapi`.

## `webui/` is generated — never edit it by hand

`task web:build` runs rsbuild with `distPath` pointed at this directory and cleans it first; the
result is committed so `go build` works without the JS toolchain. The dev server keeps its output
in memory and leaves the directory alone. `routes.json` is written by the `route-manifest` plugin
in `web/webui/rsbuild.config.ts` from the TanStack route tree.

`middleware.Static` is mounted as chi's `NotFound`, so it sees every request no API route matched:

- an existing file is served as is, `static/*` with an immutable one-year cache;
- `routes.json` itself is never served;
- anything else gets `index.html` with `no-cache` — status 200 when the path is in `routes.json`
  (or is `/`), **404 otherwise**. That includes an unknown `/api/...` path: it answers the SPA
  shell with a 404, not a JSON error.

## Things that will bite

- **`Serve` panics when `CORS` is unset.** `middleware.Cors()` returns nil without configured
  origins (its tests assert that), `router.Use(nil)` accepts it, and chi dereferences it when the
  first route is registered in `Api`. `task run` always sets `CORS`, and the tests call `Api` on a
  bare mux, so neither path hits it.
- **`Compress(5)` is router-wide but leaves SSE alone.** chi compresses only its default content
  types, and `text/event-stream` is not one of them — JSON answers and the SPA assets are.
- **One health handler, three paths.** `/health/startup`, `/health/ready` and `/health/live` are
  the same `http.Handler`; nothing here tells the probes apart.
- **Two metrics endpoints.** `/metrics` on the telemetry port is the Prometheus scrape;
  `POST /api/metrics` on the web port is the dashboard aggregate from `metrics_collector`. They
  share a name and nothing else.
- **No auth.** Neither port has any; `/api/snapshot/start` and `/api/snapshot/stop` change state.

## Tests

`webui_test.go` drives `Api` on a bare `chi.Mux` — no middleware, no static handler — with only
the operations endpoint backed by a real `operations_collector`; every other use case gets nil
dependencies, so only its request validation is reachable. The exception is `snapshot.estimate`:
its nil database answers a closed channel, so `TestSnapshotEstimateRouteStreamsEvents` gets a 200
event stream ending in `event: done`. The tests decode responses into a
client-shaped struct on purpose: they catch a field that marshals into something the generated
client cannot parse (payload as text rather than base64, duration as whole milliseconds, metadata
as `{}` rather than `null`), plus the window semantics and huma's 422s. `Serve`, the static handler
wiring, `observability.go` and `process.go` have no tests here; `middleware.Static` and `Cors` are
tested in `pkg/middleware`.

Helpers live below the last test function — see the `go` skill.
