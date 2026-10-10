# pkg/resilience — how this package hangs together

Four decorators over one function type. `Handler[T]` is "do this to one item", and `WithTimeout`,
`WithRetry`, `WithStatus` and `WithFailureMode` each wrap a handler in another handler. The package
knows nothing about sinks; `process_operation/binding.go` is the one place that composes them, once
per binding, at construction time.

## Layout

| File | Role |
|---|---|
| `handler.go` | `Handler[T]` |
| `timeout.go` | `WithTimeout` — a derived context per call |
| `retry.go` | `WithRetry`, `RetryHooks`, the pooled backoff timer |
| `backoff.go` | `Backoff` and `DelayFunc` — the linear and exponential curves, the jitter |
| `status.go` | `WithStatus`, `StatusHooks` — the terminal outcome |
| `failure.go` | `WithFailureMode`, `SkipHooks` — crash or skip |
| `errors.go` | `ErrRetryAborted` |

## The order they are composed in

From the inside out, as `newBinding` builds it:

```
WithFailureMode( WithStatus( WithRetry( WithTimeout( deliver ) ) ) )
```

Each position is deliberate, and the hooks mean what they mean only in this order:

- **`WithTimeout` is inside the retry**, so `attemptTimeout` bounds one attempt, not the delivery.
- **`WithStatus` is outside the retry**, so `OnSuccess`/`OnFailure` fire once per delivery with the
  terminal outcome, while `OnDuration` fires once per **attempt**, failed ones included.
  `adapter/metrics_collector/AGENTS.md` notes that the sink-duration metric's help text claims
  otherwise.
- **`WithFailureMode` is outermost**, so a skipped failure has already been counted by `OnFailure`
  before it is swallowed. `skip` changes what the caller sees, not what the metrics see.

The binding implements all three hook interfaces itself. A nil hooks argument is replaced by a
no-op, so the wrappers are usable bare.

## What `WithRetry` does and does not decide

- `attempts` is the **total** number of calls, not the number of retries; `0` is treated as `1`.
- It retries on **any** non-nil error. Nothing here inspects the error, so `sink.ErrSinkPermanent`
  and `template.ErrRender` are retried like everything else — see `adapter/sink/AGENTS.md`.
- The hooks carry neither the error nor the attempt index. That is why the operations log wraps the
  attempt itself (`binding.withAttempt`) instead of getting it from here.
- A cancelled context is checked before every attempt and returned raw, as is one that ends a
  backoff pause.

`delay(n)` is the wait **after** attempt `n` has failed, so the first retry waits `Initial`.

## The stop channel

`WithRetry` takes a `stop <-chan struct{}`, and the binding passes `shutdown.Draining()`. It is how
a drain cuts retries short without cancelling the context that in-flight publishes still need:

- **Closed before a pause would start** — the retries end and the last attempt's error comes back
  wrapped as `ErrRetryAborted: <error>`.
- **Closes during a pause** — the pause returns at once and **one** more attempt runs. If that fails
  too, the next pause finds the channel closed and aborts as above.
- **The first attempt always runs**, closed channel or not; stop is only consulted between attempts.
- A nil channel never closes.

`WithFailureMode` passes `ErrRetryAborted` and any error under a cancelled context through even in
`skip` mode: the item never got its full retry budget, so dropping it would lose a row the next run
could still deliver. An attempt that merely timed out is *not* such a case — the outer context is
fine, so it is skipped like any other failure.

## The timer pool

`pause` takes its timer from a `sync.Pool` and returns it without draining the channel. That is only
sound on Go 1.23+ timer semantics, where `Stop`/`Reset` guarantee no stale value is delivered; the
`go` directive in `go.mod` has to stay at 1.23 or above. `TestWithRetry_pooledTimerNotStale` and
`TestWithRetry_concurrentPauses` are the regression locks. This is what keeps the retry path
allocation-free; `WithTimeout` still allocates a context per attempt.

## Backoff

- `Type` unset or unknown falls back to **linear**. Configured backoffs never hit that, because
  `config/resilience` defaults the type to exponential.
- `Max <= 0` means no cap; `Multiplier <= 0` means 2.
- Both curves stop growing at the ceiling instead of overflowing `time.Duration`.
- **Jitter is not exponential-only**, whatever the field comment says: `DelayFunc` applies full
  jitter to whichever curve is selected, and `config/resilience` defaults `jitter` to `full` for
  every type. A linear backoff is jittered unless the config says `jitter: none`.
- Full jitter draws from `[0, delay)`, so a retry can follow almost immediately.

`config/resilience.Backoff` mirrors this struct field for field and converts with a plain
`resilience.Backoff(b)`. Adding, renaming or reordering a field here breaks that conversion — change
both declarations together.

## Tests

`countingHooks` (retry), `recordingStatusHooks` and `recordingSkipHooks` count or record hook calls.
`BenchmarkResilience` measures the composed chain with and without retries; its allocations are
expected to scale with the attempt count because of `WithTimeout`, not the retry loop.

Helpers live below the last test function — see the `go` skill.
