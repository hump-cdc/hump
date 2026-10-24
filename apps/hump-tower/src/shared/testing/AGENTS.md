# shared/testing — test doubles

Helpers that tests in every layer import. Nothing outside `*.test.*` files imports them; Biome
allows `@/shared/testing/*` from any layer. There is no barrel: import the module you need.

| Module | Holds |
|---|---|
| `api.fake.ts` | `fakeApi`, the routed `fetch` the generated client talks to (`serve`, `calls`); responses `json`, `empty`, `sse`; `createSseChannel` for a stream the test pushes into |
| `dom.fake.ts` | `registerDom` / `unregisterDom`: happy-dom on `http://localhost/`, `fakeApi` as `fetch`, a no-op `number-flow-react` element |
| `render.fake.ts` | `renderWithProviders`, `renderHookWithProviders`, `renderWithRouter` (query client without retries, tooltip provider) and `flush` |
| `browser-stubs.fake.ts` | `stubMatchMedia`, `stubLayout`, `stubResizeObserver`; each returns its restore function |
| `input.fake.ts` | `typeInto`, `openComboBox` |
| `modules.fake.ts` | `importFresh`: a new instance of a module, bypassing bun's shared cache |
| `live.fake.ts` | `fakeLiveStream`, re-exported from `shared/live/stream.fake.ts` |
| `fixtures.fake.ts` | API response bodies: `configResponse`, `metricsResponse`, `operationItem`, `operationsResponse` |

Rules:

- **Files end in `.fake.ts`, never `.fake.tsx`.** The root `bunfig.toml` keeps `**/*.fake.ts`
  out of coverage. A module that needs JSX uses `createElement`.
- **`live.fake.ts` has a side effect.** Importing it swaps the app-wide `liveStream` for the fake
  (`mock.module` in `stream.fake.ts`). A test whose import tree reaches `@/shared/live` imports
  it, by name or bare (`import '@/shared/testing/live.fake'`). The mock is process-wide, so a file
  that forgets it passes in a full `bun test` and fails when run alone.
- `fakeApi` and `fakeLiveStream` are module singletons shared by every test file: `fakeApi.serve`
  and `fakeLiveStream.reset` in `beforeEach`.
