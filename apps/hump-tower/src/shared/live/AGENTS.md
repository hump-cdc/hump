# shared/live — how this module hangs together

The client end of `GET /api/live`: one app-wide SSE connection that reconnects on its own, fans
delivery batches out to listeners, exposes the connection status and the snapshot state, and marks
the affected TanStack Query caches stale. The stream holds **no React state** — listeners are
called imperatively, so an event costs no render unless a component asked for one.

`stream.ts` holds `createLiveStream` and the `liveStream` singleton; `LiveUpdates` turns events
into invalidations; the hooks `useLiveOperations` (no re-render), `useLiveStatus` and
`useLiveSnapshot` (`useSyncExternalStore`) are the consumer API. `stream.fake.ts` is test-only
and not exported from `index.ts`.

## Who holds the connection

Only `LiveUpdates` calls `retain()`; **the hooks subscribe and nothing else**. It is mounted once,
in `src/app/providers.tsx`, inside `QueryClientProvider`. A hook used in a tree without it sees
`idle` forever and never gets a batch.

`retain` is reference-counted and returns an idempotent release. When the last hold goes, the
connection lingers `LINGER_MS` (1s) before `stop()`, and a `retain` inside that window cancels the
timer — that is what lets a StrictMode remount (`app/main.tsx`) reuse the connection instead of
opening a second one. `stop()` zeroes the holds, aborts, drops the snapshot and goes `idle`; it is
for teardown (tests, and the `import.meta.webpackHot` dispose at the bottom of `stream.ts`), not
for consumers. It does not clear the listener sets.

## One attempt, and why reconnecting lives here

`openLiveStream` calls the generated `Live.stream` with `sseMaxRetryAttempts: 1`, so the transport
(`shared/api/generated/core/serverSentEvents.gen.ts`) makes exactly one attempt. Its own retry loop
is switched off because it never resets its backoff, stops for good on a clean end, and sleeps the
whole backoff before noticing an abort.

The transport catches its own failures — a failed fetch, a non-OK status, a payload the schema
rejects — reports them through `onSseError` and **ends the iteration instead of throwing**. So in
production a broken attempt reaches `run` as a clean end, and `[live] stream error` in the console
is the only trace; the `catch` in `run` is for an `open` that does throw. Both paths, and a normal
end, lead to the same place: unless the signal is aborted, retry. A server restart looks like a
clean end, which is why that is never terminal.

Delays come from `createBackoff`: `BACKOFF_BASE_MS` (1s) doubling per attempt up to
`BACKOFF_MAX_MS` (30s). An attempt that lived `STABLE_ATTEMPT_MS` (10s) or longer resets it.
The wait is the abortable `sleep`, so `stop()` does not have to sit out a backoff.

## Status

`idle → connecting → live → reconnecting → connecting → …`, back to `idle` on `stop`.

- `live` is set by **any** event. The server sends `welcome` as soon as it accepts the connection,
  so `connecting` does not wait for traffic.
- `reconnecting` is set when an attempt ends, and lasts for the backoff only: the next attempt
  flips to `connecting` again.
- `setStatus` and `setSnapshot` skip a value equal to the current one, so listeners fire on real
  transitions only. The snapshot is compared by `state` and `progress`.
- `run`'s `finally` resets `controller` and the status only if it still owns the controller, so an
  attempt that outlives a `stop()` + `retain()` cannot clobber its successor.

## What an event turns into

The loop classifies each payload by shape (see "The wire contract"):

| Payload | Effect |
|---|---|
| has `state` | stored as the snapshot, snapshot listeners notified; **not** an operations event |
| has a non-empty `operations` | operations listeners get the array |
| anything else (`welcome`, empty or `null` `operations`) | only proves the connection is up |

`notify` iterates a copy of the set and wraps each call in `try`/`catch`: a listener may
unsubscribe from inside itself, and one that throws is logged without starving the others or
ending the stream.

The snapshot is **dropped to `undefined` whenever an attempt ends** (and on `stop`). The server
reports it again on every connect, so a value from a dead connection is never shown next to an
offline badge.

Reconnect listeners fire at the start of every attempt after the first, before `connecting`.
Events in the gap are lost for good, so this is the signal to refetch.

`LiveUpdates` maps the two signals onto `shared/state/invalidate.ts`:

- a batch → `invalidateForLiveEvent`: the `metrics` key prefix;
- a reconnect → `invalidateForLiveReconnect`: `metrics` with `refetchType: 'all'`, plus the
  `operations` prefix.

Both pass `cancelRefetch: false`. Snapshot events invalidate nothing. Which queries a signal
touches is decided in `shared/state`, not here.

## The wire contract

Server side: `apps/hump/internal/application/hump_tower/stream_updates/AGENTS.md` (the route, the
event order, the snapshot dedupe), `apps/hump/internal/adapter/live_streamer/AGENTS.md` (the
batching window, lossy fan-out), `apps/hump/internal/port/live_stream/AGENTS.md` (why an operation
is only schema, table, sink, host, target).

The server names its events `welcome`, `operations`, `snapshot`, but **the client never sees the
names**: the spec flattens them into a union of payloads (see "SSE responses are unions without
event names" in `shared/api/AGENTS.md`), so events are told apart by shape — `isSnapshot` is
`'state' in event`, operations is `'operations' in event`.

When the backend adds an event or changes a payload:

1. Backend: `dto.go`, `Events()` and the `flattenSSESchema` list for `/api/live`; then
   `moon run hump-tower:openapi` and `bun run generate:openapi` regenerate the spec and the client.
2. Here: add a branch in the `run` loop **before** the operations fallthrough — an unhandled
   payload is silently treated as an empty batch. A new kind of data also needs its alias in
   `types.ts`, a listener set with `subscribe*` / `get*` on `LiveStream`, a hook, and the exports
   in `index.ts`.
3. No new payload may carry `state` or `operations`. The `in` checks are string literals, so
   renaming either field compiles and silently reroutes the event.

Renaming only an event name changes nothing here. A payload the schema rejects — a snapshot with
an unknown `state`, say — is not skipped: it ends the attempt and the stream reconnects.

`/api/snapshot` is **not** read by this module. `features/snapshot/model/snapshot.ts` opens it
through `Snapshot.stream` with its own retry; see
`apps/hump/internal/application/hump_tower/stream_snapshot/AGENTS.md`. `/api/live` carries only the
state and the whole percent, enough for the badge every page shows.

## Tests

- The hook tests and `live-updates.test.tsx` render against `fakeLiveStream` from
  `stream.fake.ts`: a hand-driven `LiveStream` (`emit`, `setStatus`, `setSnapshot`, `reconnect`)
  that also counts holds and listeners. Importing `stream.fake.ts` swaps the `liveStream`
  singleton through `mock.module('./stream', …)` and leaves the rest of the module real. The mock
  is process-wide, hence one shared fake that each test `reset()`s.
- Those files call `registerDom` / `unregisterDom` (happy-dom) in `beforeAll` / `afterAll`.
  There is deliberately no preload: happy-dom replaces `fetch`, and the integration test below
  needs the runtime's own.
- `stream.test.ts` drives `createLiveStream` through `LiveStreamDeps`: `open` returns a
  test-driven `Channel` per attempt (`push` / `end` / `fail`), `sleep` records the delay and
  advances a fake clock instead of waiting, and the harness sets `lingerMs` to 0. `tick()` flushes
  microtasks and zero-delay timers; the retain tests use real delays.
- `stream.integration.test.ts` runs the generated `Live.stream` against a `Bun.serve` on port 0.
  It pins what `stream.ts` relies on from the transport: named frames arrive as bare payloads, a
  server close ends the iteration, and a rejected payload yields nothing and calls `onSseError`
  once. It needs the generated client, and no backend.
