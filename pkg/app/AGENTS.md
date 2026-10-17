# pkg/app — how this package hangs together

The process lifecycle. `app` owns the runner `cmd/hump/main.go` hangs every long-lived function on,
`app/options` parses the command line into the one `Options` value, and `app/shutdown` is the
process-global registry of teardown hooks that every adapter registers itself with.

## Layout

| File | Role |
|---|---|
| `runner.go` | the `runner`: `New`, `Run`, `Serve`, `Wait` |
| `signal.go` | `signals` — a context cancelled by SIGINT/SIGTERM, with the signal as its cause |
| `errors.go` | `ErrShutdown`, `ErrDrainTimeout` |
| `options/types.go` | the `Options` struct (go-flags tags) and the config errors |
| `options/parser.go` | `Parse` and the config source resolution |
| `shutdown/shutdown.go` | the stages and the global API: `OnShutdown`, `Shutdown`, `Drain`, `Draining` |
| `shutdown/registry.go` | the registry: stage ordering, the per-hook timeout |
| `shutdown/errors.go` | `ErrShutdownTimeout` |

## Workers and servers are different things

`Run` launches **workers**, `Serve` launches **servers**, and `Wait` treats them differently:

1. Block until the runner's context is done — a signal, or any function returning a non-nil error.
2. `release()` the signal hook, so a second signal kills the process the default way.
3. `shutdown.Drain()`, then join the workers, for at most `drainTimeout` (30s). Running out records
   `ErrDrainTimeout`, which makes the exit non-zero, and carries on.
4. `shutdown.Shutdown(hookTimeout)` — every hook, 5s each.
5. Join the servers, for at most `hookTimeout`. Running out only logs.
6. Return the recorded errors joined.

So for a worker, a cancelled context means **"take no new work"**, not "abort": it is still expected
to finish what it picked up, and nothing is torn down under it until it returns or 30s pass. A
server is the opposite — it is stopped by its own `StageServers` hook, which it registers from inside
its `Serve` function. `ObservabilityController.Serve` does not even take the context. The bounded join
in step 5 exists for a server that died or never got as far as registering that hook.

What ends up in the result:

- Only errors **returned by `Run`/`Serve` functions** are recorded. A function returning `nil` just
  ends; it does not stop the runner.
- `result` drops anything that `errors.Is` `context.Canceled` or `ErrShutdown`. Returning
  `app.ErrShutdown` is how a function asks for a clean exit — `--cleanup` in `main.go` does exactly
  that.
- The signal itself is a cancellation *cause*, never a recorded error, so a plain Ctrl-C exits zero.
- Hook errors are logged by the registry and then discarded (`_ = r.shutdown(...)`). A failing or
  timed-out hook does not change the exit code.

`New` returns the unexported `*runner`; callers hold it by inference and nothing names the type.

## The shutdown registry

One process-global registry, three stages, run in this order:

| Stage | Who registers there |
|---|---|
| `StageWorkers` | `process_operation` (the worker pool), `wal_position`, `snapshot`, the three dashboard collectors |
| `StageSinks` | every cached sink client — Kafka, NATS, RabbitMQ, RabbitMQ Streams |
| `StageServers` | the web UI and telemetry HTTP servers |

Every hook of a stage finishes before the next stage starts, and **within a stage hooks run in
reverse registration order**. That reverse order is load-bearing — `adapter/live_streamer/AGENTS.md`
depends on it to tear the streamer down before the collector it wraps. Registration order is
whatever order the constructors happen to run in, so hoisting a `New` call can invert a teardown
silently.

Things that follow from `registry.shutdown`:

- It runs **once**. The handler list is taken and cleared under the lock, so a hook registered after
  shutdown started is never called. Sink clients are created lazily on first publish; one created
  that late is simply not closed.
- Each hook gets its own `context.WithTimeout(context.Background(), timeout)`. On timeout the
  registry logs, returns `ErrShutdownTimeout` for that hook and moves on — the hook's goroutine is
  abandoned, not cancelled, unless the hook honours its context.
- An error from one hook never stops the others.

## Drain is a one-way switch

`Drain` closes a package-level channel once; `Draining` hands that channel out. Its only reader is
`resilience.WithRetry`, which takes it as the stop channel (`process_operation/binding.go`), so a
delivery mid-backoff makes one last attempt instead of sleeping through the drain window — see
`pkg/resilience/AGENTS.md`.

There is no way to reopen it. `TestDrainClosesDraining` closes it for the rest of its test binary,
and any other package that called `shutdown.Drain()` in a test would do the same to its own.

## Options

`Parse` reads `os.Args` through go-flags, then `resolveConfig` picks the config text:

1. `--config` and `--config-file` together are `ErrConfigConflict`.
2. Either flag wins over the environment.
3. Otherwise `CONFIG`, then `CONFIG_FILE` — both set means `CONFIG` wins, silently.
4. Otherwise stdin, but only when it is not a character device (a pipe or a redirect).
5. Nothing found is `ErrConfigMissing`.

`CONFIG` and `CONFIG_FILE` are read by hand in `resolveConfig`, not through `env:` tags, which is
why the help text spells them out in the description — and `--config`'s says `[$CONFG]`, a typo.
After `Parse`, `Options.Config` always holds the config **text**, whichever source it came from.

`TelemetryPort` and `WebPort` are `*uint`: nil means the server is off, and `main.go` skips the web
UI adapters entirely when `WebPort` is nil.

`Options` is not private to `main`: `webui/get_config` receives it by value to show the effective
settings, and `cmd/openapi` passes a zero `Options{}`. A new field is visible to both.

## Tests

`newTestRunner` shortens both timeouts and swaps the `drain` and `shutdown` fields for no-ops, which
is the only reason those are fields at all — a runner test must stay off the global registry and
must not close `Draining`. `TestGlobalShutdownRunsRegisteredHandlers` swaps `globalRegistry` for the
same reason; the other registry tests build a `registry{}` of their own.

The options tests mutate `os.Args`, the environment and `os.Stdin` through `setup` and `setStdin`,
so they cannot run in parallel. `setup` points stdin at `/dev/null` on purpose: it is a character
device, which `Parse` does not read config from.

Helpers live below the last test function — see the `go` skill.
