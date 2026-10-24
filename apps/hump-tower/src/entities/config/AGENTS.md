# entities/config — how this module hangs together

FSD `entities` layer. Three slices that describe the running server to the rest of the UI:
`config` (the `GET /api/config` query and the gate around the app), `source-viewer` (the raw config
text) and `state` (the header's live controls and the chart step). Nothing here writes to the
backend.

There is no `entities/config/index.ts`; import each slice by its full path.

## Public contract

| Barrel | Exports | Consumed by |
|---|---|---|
| `config` | `useConfig`, `ConfigProvider` | `src/app/providers.tsx`, `widgets/sidebar/connection-info.tsx` |
| `source-viewer` | `SourceViewer` | `pages/configuration` |
| `state` | `LiveControls` | `widgets/sidebar/app-header.tsx` |

Other entities use the `@x` files: `config/@x/<consumer>.ts` re-exports `useConfig`,
`state/@x/<consumer>.ts` re-exports `useLiveControlsChartStep`. `useLiveControlsChartStep` is
exported **only** through `@x`; the store and `CHART_STEPS` stay inside the slice.

## `config`: fetched once, then a constant

`useConfig` spreads the generated `getApiConfigOptions()` and sets `staleTime` and `gcTime` to
infinity. The backend builds the response from what the process was started with
(`apps/hump/internal/application/hump_tower/get_config/AGENTS.md`), so there is nothing to refresh.

- The query key is the generated one (`[{ _id: 'getApiConfig', baseUrl }]`), not an entry in
  `shared/state` `keys`, so neither a live event nor a reconnect touches the config.
- Every consumer calls `useConfig()` with no arguments and shares one cache entry. The generated
  key includes any `query`, `headers`, `path` or `baseUrl` option, so a variant with options would
  be a second entry and a second request.

`ConfigProvider` is a **gate, not a context provider**: pending → `ConfigLoading`, error →
`ConfigError`, otherwise `children` (placement: `src/app/AGENTS.md`). Everything below it renders
with the config in the cache, but TypeScript does not know that: `data` is still `T | undefined`,
so consumers write `config?.tables ?? []` or check `isSuccess`. `sinks` and `tables` are also
typed `| null`.

`ConfigError` fills in what the `ErrorModelModel` lacks: status `500`, title
`Something went wrong`, and a fixed sentence for an empty `detail`.

## `source-viewer`

Renders `rawConfig`: the configuration exactly as passed at startup, not a re-marshalled one.
`rawConfig` and `configPath` are optional in the response: the text falls back to `''` and the
path is not rendered.

## `state`: the chart step

`useLiveControlsStore` is a zustand store persisted to `localStorage` under `hump.live-controls`;
`partialize` keeps `chartStep` only. Default `1m`.

The step is not a display setting, it is the request. Every consumer passes it to
`useMetrics(chartStep)` (`entities/metrics`), where it becomes the `duration` of the
`/api/metrics` body and part of the query key. So:

- all charts, the flows canvas, the inspector and the operations filter share **one** metrics
  query; switching the step switches all of them to another entry;
- `CHART_STEPS` is declared with `satisfies readonly MetricsRequestBodyModel['duration'][]`, so a
  step the API does not accept fails to compile. The check is one-way: a width added on the
  backend does not appear in the dropdown until it is added to the list;
- the store is at `version: 1`, and `migrateLiveControls` keeps a stored step only if it is still
  in `CHART_STEPS`. zustand discards stored state on a version mismatch when there is no
  `migrate`, so a version bump means extending that function.

## `state`: the two badges

`LiveControls` returns a fragment; the parent's flex row in `app-header.tsx` lays it out.

- `LiveStatus` is the `asChild` trigger of the dropdown, so it must keep spreading its props onto
  `Badge`. `idle` and `reconnecting` both read `offline`.
- `SnapshotStatus` renders only while the snapshot state is `processing`, then links to
  `/snapshot`. Its percent comes from the `snapshot` event of `/api/live` (`useLiveSnapshot`), not
  from `/api/snapshot`.
