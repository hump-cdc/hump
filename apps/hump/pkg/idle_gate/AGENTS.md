# pkg/idle_gate — how this package hangs together

A refcount with a grace period. A `Gate` runs `onStart` on the first `Connect` and `onStop` once no
client has been connected for `timeout`, so a resource nobody is watching costs nothing. It is what
keeps `adapter/metrics_collector` and `adapter/operations_collector` idle until a dashboard opens,
and both of their AGENTS.md files rely on the guarantees below.

## The contract callers build on

- **Callbacks run under the gate's lock.** `onStart` and `onStop` are called with `g.mu` held, so
  neither may call `Connect`, `Disconnect` or `Close` — that deadlocks. A callback that takes a lock
  of its own fixes the order as gate-then-callback, and every other path in that package has to
  respect it: nothing holding the callback's lock may reach the gate.
- **The active flag is cleared after `onStop` returns** (and set after `onStart` returns). A caller
  that sees `Active() == false` therefore also sees whatever `onStop` tore down;
  `operations_collector` turns that into "`Connected() == false` implies `ring == nil`".
  `TestGateStopsCallbackBeforeClearingActive` pins the order.
- **`Active` is one atomic load and takes no lock**, because the pipeline calls it per delivery. It
  is a hint: the last client can leave right after it returns true.
- **`Close` is final and idempotent.** It calls `onStop` only if the gate was active, and a later
  `Connect` or `Disconnect` does nothing.
- **An unpaired `Disconnect` is ignored** rather than driving the count negative.

## `generation`, and the race it exists for

`Disconnect` arms a `time.AfterFunc` whose closure carries the generation it was armed under, and
`Connect` bumps the generation. Stopping the timer in `Connect` is not enough on its own: a timer
that has already fired and is parked on `g.mu` cannot be stopped. Without the generation check —
client leaves, timer T1 fires and blocks on the lock, a client connects and leaves again arming T2,
T1 gets the lock with zero connections and deactivates a gate whose grace period has only just
started. `deactivate` returns early on a stale generation, on a non-zero count, and on a closed gate.

`adapter/live_streamer` uses the same fix for the same shape of bug in its window timer.

## No goroutine of its own

Between a `Disconnect` and the timeout the only thing alive is the runtime timer; an idle or a
connected gate holds nothing. `TestGateHasNoBackgroundGoroutines` checks that with goleak, and
`operations_collector`'s goleak test depends on it (`metrics_collector` has none).

## Tests

`newTestGate` returns a gate with a `callCounter` for callbacks and registers `Close` as cleanup.
`callCounter` has its own mutex because the idle timer calls `onStop` from a goroutine of its own.
`waitFor` polls a condition for a second.

`TestGateIgnoresStaleDeactivation` calls `deactivate` with an old generation directly, since the
real race is not reproducible on demand.
