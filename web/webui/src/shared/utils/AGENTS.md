# shared/utils — how this module hangs together

Small helpers with no knowledge of the domain: formatters, a few hooks, `cn`, `debounce`, `sleep`.
FSD layer: `shared`. Nothing here imports from the app at all — not even another `shared/*`
segment; dependencies are `react`, `@tanstack/react-router` (one hook), `clsx`, `tailwind-merge`.

Imported, always through `@/shared/utils`, by `shared/ui` (mostly `cn`), `shared/live` (`sleep`),
`entities/{config,flows,operations,statistics}`, `features/snapshot`, `pages/{flows,operations}`
and `widgets/sidebar`. `index.ts` is `export *` of every file, so anything a file exports is
public.

## File naming

- `format.<what>.ts` — pure formatters, `number`/`Date` in, `string` out, plus the `scale*`
  variants below. Formatter instances (`Intl.*`) are created once at module level.
- `use-<name>.ts(x)` — one React hook per file. `use-switch-animation.ts` also exports
  `useSwitchViewValue` and `SwitchViewValueContext`, which go with `useSwitchAnimation`.
- A bare name (`cn.ts`, `debounce.ts`, `sleep.ts`) — everything else. `debounce.ts` holds both
  the plain function and its hook, `useDebounce`.
- Tests sit next to the file as `<name>.test.ts` on `bun:test`. `debounce` and `sleep` have one.

## Formatters

All output is fixed to English locales (`en-US`, `en-GB`), independent of the browser's. Thousands
are grouped with a space, not a comma: `formatNumber*` and `formatNumberCompact` use a
**non-breaking space** (U+00A0), `formatBytes` a plain space (U+0020): `1 284 300`.

| Helper | Input | Output and guarantees |
|---|---|---|
| `formatNumber(v, suffix?)` | number | grouped integer, rounded: `1 284 300`; suffix appended with no space |
| `formatNumber1` / `formatNumber2` | number | same, with exactly 1 / 2 fraction digits: `12.5%`, `12.50%` |
| `formatNumberCompact(v)` | number | `3.04M` — compact notation, at most 2 fraction digits |
| `formatBytes(v)` | bytes | 1024-based, `B`…`TB`: `512 B`, `1.50 KB`; integer for `B`, 2 digits otherwise; keeps the sign; stops at `TB` |
| `scaleBytes(v)` | bytes | `{ value, suffix, fractionDigits }` for the same scale; suffix has a leading space (`' KB'`) |
| `formatLatency(ms)` | **milliseconds** | `12ms` below 1000, else seconds with 2 digits: `1.24s`; negatives clamp to 0 |
| `scaleLatency(ms)` | milliseconds | `{ value, suffix, fractionDigits }`; suffix has no space (`'ms'`, `'s'`) |
| `formatDuration(s)` | **seconds** | `48s`, `1m 48s`; rounded to whole seconds, negatives clamp to 0, no hours unit |
| `formatDateTime(d)` | `Date` | `Intl.DateTimeFormat('en-GB')` output, local time: `17 Oct 2026, 14:03:05` in Node, `17 Oct 2026 at 14:03:05` in Bun; the separator is engine-dependent |
| `formatDateTimeRange(a, b)` | two `Date`s | `formatDateTime(a) → HH:mm:ss`, same separator caveat; the end repeats the date only when it is another local day |
| `formatTimeCompact(ts)` | epoch ms | zero-padded local `HH:MM`, for chart axes |
| `formatTime(d)` | `Date` | `HH:mm:ss.SSS`, 24-hour, local time |

- `formatNumber*`, `formatNumberCompact` and `formatBytes` are generic over `number | undefined`:
  `undefined` in gives `undefined` out, and the return type follows the argument, so a call with a
  definite number needs no `??`. The other formatters take a definite value.
- The `scale*` pair exists for `shared/ui/number-flow`, which animates the number and needs the
  value and its unit apart. They must stay consistent with `formatBytes`/`formatLatency`; the
  thresholds are written twice in each file.
- The millisecond boundary compares before it rounds: `formatLatency(999.6)` stays in the `ms`
  branch and prints `1000ms`.
- No formatter sets a `timeZone`: every date and time is printed in the browser's local time.

## Hooks and the rest

| Helper | What it guarantees |
|---|---|
| `cn(...inputs)` | `clsx` then `tailwind-merge`; the merge is extended to know `shadow-glow` (the custom `--shadow-glow` in `app/styles.css`) as a shadow class |
| `debounce(fn, ms, { maxWait })` | trailing edge only, called with the last arguments; with `maxWait`, runs at least that often counted from the first call of a burst; `.cancel()` drops the pending call |
| `useDebounce(fn, ms, options)` | a `debounce` with a stable identity that always calls the latest `fn` and cancels on unmount; `ms` and `options` are read on the first render only |
| `sleep(ms, signal?)` | resolves after `ms`, as soon as `signal` aborts, or at once when it is already aborted — it **never rejects**, so the caller checks `signal.aborted` itself |
| `useClipboard(resetDelay = 1500)` | `copy(text)` writes to the clipboard; `copied` is true for `resetDelay` ms after a success; a failed write is swallowed |
| `useElementSize()` | `[callbackRef, { width, height }]` from `offsetWidth`/`offsetHeight`, kept current by a `ResizeObserver`; zero while no element is attached |
| `useIsMobile(breakpoint = MOBILE_BREAKPOINT)` | true below 1024 px (`max-width: breakpoint - 1`), tracked through `matchMedia` |
| `useStaticData()` | the `staticData` of the deepest matched route that has a `title`, or `{}` |
| `useSwitchAnimation(value)` | `{ current, leaving, views, onLeaveEnd }`: keeps the previous value mounted next to the new one for a cross-fade |
| `useSwitchViewValue<T>()` | the value of the enclosing `SwitchViewValueContext`; `T` is asserted, not checked |

Things that are easy to get wrong:

- **`use-static-data.tsx` is also a type declaration.** It augments TanStack Router's
  `StaticDataRouteOption` with `title` and `headerAside`; that is what makes `staticData` typed in
  the route files under `app/routes`. A new static field for routes is added there. Because the
  lookup keys on `title`, a route that sets `headerAside` without `title` contributes nothing.
- **`useSwitchAnimation` does not time anything.** `leaving` stays set — and the old view stays
  mounted — until the consumer calls `onLeaveEnd` (from its animation-end handler). A change
  mid-animation replaces the leaving value. The consumer provides `SwitchViewValueContext` per
  view, so a leaving view can keep rendering the last value it had (`features/snapshot`).
- `useDebounce` creates its timer once; changing `ms` between renders has no effect.

## What belongs here

A helper goes here when it knows nothing about hump — no API types, no entity, no query keys —
and renders nothing. Otherwise:

- it renders, or exists for one component → `shared/ui/<component>` (see `shared/ui/AGENTS.md`);
- it formats or computes something of a domain type → the `model/` of the owning entity;
- it touches the query client or keys → `shared/state`; the live connection → `shared/live`.

Add the file under the naming scheme above and add its `export *` line to `index.ts`, which is
kept alphabetical.
