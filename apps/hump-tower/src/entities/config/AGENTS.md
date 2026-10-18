# entities/config — how this module hangs together

FSD `entities` layer. Three slices that describe the running server to the rest of the UI: the
backend configuration (`GET /api/config`), a read-only view of its raw text, and the header's live
controls. Everything here is read-only towards the backend — there is no mutation.

There is **no `entities/config/index.ts`**: each slice has its own barrel and is imported by its
full path — `@/entities/config/config`, `@/entities/config/state`,
`@/entities/config/source-viewer`. The only test is `state/model/store.test.ts`; there are no
stories.

## Layout

| Path | Role |
|---|---|
| `config/model/use-config.ts` | `useConfig`: the one `/api/config` query |
| `config/ui/config-provider.tsx` | `ConfigProvider`: gate that renders children only once the config is loaded |
| `config/ui/config-loading.tsx`, `config-error.tsx` | the gate's two other screens; not exported |
| `config/@x/*.ts` | `useConfig` for same-layer consumers |
| `source-viewer/ui/source-viewer.tsx` | `SourceViewer`: raw config text with its path and a copy button |
| `state/model/store.ts` | `useLiveControlsStore`, `CHART_STEPS`, `ChartStep`, `useLiveControlsChartStep`, `migrateLiveControls` |
| `state/ui/live-controls.tsx` | `LiveControls`: snapshot badge + live badge with the chart-step dropdown |
| `state/ui/live-status.tsx`, `snapshot-status.tsx` | the two badges; not exported |
| `state/@x/*.ts` | `useLiveControlsChartStep` for same-layer consumers |

## Public contract

| Barrel | Exports | Consumed by |
|---|---|---|
| `config` | `useConfig`, `ConfigProvider` | `src/app/providers.tsx`, `widgets/sidebar/connection-info.tsx` |
| `source-viewer` | `SourceViewer` | `pages/configuration` |
| `state` | `LiveControls` | `widgets/sidebar/app-header.tsx` (`LiveControls`) |

Other entities may not use the barrels; they go through `@x`, one file per consumer, each
re-exporting the single thing that consumer needs:

| File | Exports | Consumer |
|---|---|---|
| `config/@x/statistics.ts` | `useConfig` | `statistics/tables`, `statistics/sinks` |
| `config/@x/operations.ts` | `useConfig` | `operations/filter` |
| `config/@x/inspector.ts` | `useConfig` | `flows/inspector` |
| `config/@x/source-viewer.ts` | `useConfig` | `source-viewer` (the sibling slice of this module) |
| `state/@x/statistics.ts` | `useLiveControlsChartStep` | the seven `statistics/*` chart slices |
| `state/@x/flows.ts` | `useLiveControlsChartStep` | `flows/flows` |
| `state/@x/inspector.ts` | `useLiveControlsChartStep` | `flows/inspector` |
| `state/@x/operations.ts` | `useLiveControlsChartStep` | `operations/filter` |

`useLiveControlsChartStep` is exported **only** through `@x`, not through `state/index.ts`; the
store itself and `CHART_STEPS` stay inside the slice. A new
same-layer consumer gets its own `@x/<consumer>.ts`; the `src/entities/**` override of
`noRestrictedImports` in `biome.json` rejects every other `@/entities/**` import from an entity.

The module itself imports no other entity. From `shared` it uses `shared/api`
(`getApiConfigOptions`, `ErrorModelModel`, `MetricsRequestBodyModel`), `shared/live` (`useLiveStatus`, `useLiveSnapshot`),
`shared/utils` (`cn`, `useClipboard`) and these `shared/ui` components: `badge`, `button`, `code`,
`dropdown-menu`, `error`, `number-flow`, `tabs`. It does not import `shared/state`.

## `config`: fetched once, then a constant

`useConfig` spreads the generated `getApiConfigOptions()` and sets `staleTime` and `gcTime` to
infinity. The backend builds the response from what the process was started with (see
`apps/hump/internal/application/hump_tower/get_config/AGENTS.md`), so there is nothing to refresh.

- The query key is the generated one (`[{ _id: 'getApiConfig', baseUrl }]`), not an entry in
  `shared/state` `keys`. `LIVE_EVENT_QUERIES` / `LIVE_RECONNECT_QUERIES` only cover the `metrics`
  and `operations` prefixes, so neither a live event nor a reconnect touches the config.
- Every consumer calls `useConfig()` with no arguments and shares that one cache entry. The
  generated key includes any `query`, `headers`, `path` or `baseUrl` option, so a variant with
  options would be a second entry and a second request.
- The generated SDK parses the response with `zGetApiConfigResponse` before it reaches the cache.

`ConfigProvider` is a **gate, not a context provider**: pending → `ConfigLoading`, error →
`ConfigError`, otherwise `children`. `src/app/providers.tsx` wraps the whole route tree in it, so
everything below renders with the config already in the cache. TypeScript does not know that —
`data` is still `T | undefined` — which is why consumers write `config?.tables ?? []` or check
`isSuccess`. `sinks` and `tables` are also typed `| null`.

`LiveUpdates` and `Toaster` are mounted **outside** the gate in `providers.tsx`: the live stream
connects whether or not the config loaded.

`ConfigError` takes the `useConfig()` query's error (an `ErrorModelModel`) and fills in what is missing: status `500`,
title `Something went wrong`, and a fixed sentence for an empty `detail`.

## `source-viewer`

Renders `rawConfig` in `shared/ui/code` — the configuration exactly as it was passed at startup,
not a re-marshalled one. Both `rawConfig` and `configPath` are optional in the response: the text
falls back to an empty string and the path is simply not rendered. The copy button copies the
same string that is shown.

## `state`: the chart step

`useLiveControlsStore` is a zustand store persisted to `localStorage` under `hump.live-controls`;
`partialize` keeps `chartStep` only. Default `1m`.

The step is not a display setting — it is the request. Every consumer passes it straight to
`useMetrics(chartStep)` (`entities/metrics`), where it becomes the `duration` of the `/api/metrics`
body and the last element of `keys.queries.metrics(duration)`. So:

- all charts, the flows canvas, the inspector and the operations filter read **one** metrics query
  as long as they use the same step; switching the step switches all of them to another entry;
- `CHART_STEPS` is declared with `satisfies readonly MetricsRequestBodyModel['duration'][]`, so a
  step the API does not accept fails to compile right there. The check is one-way: a width added
  on the backend does not appear in the dropdown until it is added to the list;
- the store is at `version: 1`, and `migrateLiveControls` keeps a stored step only if it is still
  in `CHART_STEPS`, falling back to `1m`. zustand discards stored state on a version mismatch when
  there is no `migrate`, so a version bump means extending that function. `migrate` runs on a
  mismatch only — a value written at the current version is restored as is.

## `state`: the two badges

`LiveControls` returns a fragment — the snapshot badge and the dropdown are laid out by the
parent's flex row in `app-header.tsx`.

- `LiveStatus` is the `asChild` trigger of the dropdown, so it must keep spreading its props onto
  `Badge`. `STATUS_BADGES` is a `Record<LiveStatus, …>` over `shared/live`'s status union: `idle`
  and `reconnecting` both read `offline`, `connecting` and `live` have their own.
- `SnapshotStatus` renders nothing unless the snapshot state is `processing`, then links to
  `/snapshot`. Its percent comes from the `snapshot` event of `/api/live`
  (`apps/hump/internal/application/hump_tower/stream_updates/AGENTS.md`) through `useLiveSnapshot`,
  not from `/api/snapshot`.
