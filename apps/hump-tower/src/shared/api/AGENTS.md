# shared/api — how this module hangs together

The typed HTTP client of the web UI. Almost all of it is generated from the OpenAPI spec the Go
side emits; the hand-written part is `client.ts` (base URL), `error-message.ts` (`ApiError`,
`errorMessage`) and `index.ts`. It imports nothing from another `shared/*` segment.

`index.ts` re-exports `error-message`, `generated/index.ts` (the SDK classes, `Options`, every
`*Model` type), `generated/@tanstack/react-query.gen` and `generated/zod.gen`. `client.ts` and
the `client` instance are not exported: the base URL is set in one place and nowhere else.

## The generation pipeline

```
apps/hump/internal/application/hump_tower/*/dto.go + controller.go   (huma operations)
    → moon run hump-tower:openapi    go -C ../hump run ./cmd/openapi, OPENAPI_OUT=apps/hump-tower/src/shared/api/openapi.json
    → bun run generate:openapi    openapi-ts, config in apps/hump-tower/openapi-ts.config.ts
    → generated/
```

- `apps/hump/cmd/openapi/main.go` builds the same `controller.NewHumpTowerController` the server
  uses, with empty dependencies (handlers are registered, never called), and marshals
  `api.OpenAPI()`. It fails without `OPENAPI_OUT`, which only the `openapi` task sets.
- `predev` / `prebuild` regenerate the client from whatever `openapi.json` is on disk; they do
  **not** refresh the spec. After changing a Go DTO or route, run `moon run hump-tower:openapi`
  (or `hump-tower:dev` / `hump-tower:build`, which depend on it).
- A change to the API shape is made in Go; a change to the generated code's form is made in
  `openapi-ts.config.ts`.

What `openapi-ts.config.ts` decides, and where it shows:

| Plugin setting | Effect |
|---|---|
| `client-fetch` `runtimeConfigPath` | `generated/client.gen.ts` imports `createClientConfig` from `../client.ts` |
| `typescript` `definitions.name` | schema types are suffixed: `ErrorModel` → `ErrorModelModel` |
| `sdk` `byTags` + `nesting` | a static class per tag, method = operationId minus its first segment: `snapshot.stream` → `Snapshot.stream` |
| `sdk` `transformer: true` | every response with a body goes through its zod schema (`responseTransformer`) |
| `@tanstack/react-query` | `getApiConfigOptions`/`getApiConfigQueryKey` for the one plain GET; `post*Mutation` for POSTs, except the SSE `POST /api/snapshot/estimate` |
| `zod` | `<Name>Schema` + inferred type `<Name>`, plus per-operation `zGetApi…` / `zPostApi…` request and response schemas |

Two things follow. Responses are **parsed at runtime** (`parseAsync`), not only typed. And the
reads that are POSTs (`/api/metrics`, `/api/operations`, `/api/snapshot/tables`) get mutation
options from the generator, not query options — their consumers write `useQuery` by hand around
the SDK call, with keys from `shared/state`. Only `entities/config` uses a generated options
helper.

## Base URL

`client.ts` resolves it once, at module load: `API_URL` if non-empty, else `location.origin`,
else `''`. The `typeof API_URL` guard covers runs where nothing defines the constant, such as
`bun test`. In dev the UI (port 3000) calls the API on port 3200 cross-origin;
`moon run hump:run` sets `CORS=http://localhost:3000` for `pkg/middleware.Cors` to allow it.

## Errors

SDK methods default to `throwOnError: false` and return `{ data, error }`. Every request/response
call in the app, and every generated query/mutation option, passes `throwOnError: true`. What is
thrown then is the parsed `application/problem+json` body — a plain object, **not an `Error`** —
hence `ApiError = Error | ErrorModelModel`.

`error-message.ts` registers `ApiError` as react-query's `defaultError` by module augmentation,
so `error` from `useQuery`/`useMutation` is typed app-wide. `errorMessage(error)` picks, in order:
`Error.message`; for a problem body `detail`, then `title`, then `status`, then
`'Unknown error'`; anything else `String(error)`. `isErrorModel` checks the shape at runtime
because the client throws the body untyped. Use `errorMessage` for anything shown to the user.

## SSE responses are unions without event names

Three routes stream (`Live.stream`, `Snapshot.stream`, `Snapshot.estimate`). huma documents an SSE
response as an array of `{data, event, id, retry}` envelopes, but the generated client applies the
response schema to the `data` of a single event. `flattenSSESchema` in
`apps/hump/cmd/openapi/main.go` therefore replaces each of the three response schemas with a `oneOf`
of the event payloads, which is why `GetApiLiveResponse` is
`OperationsEventModel | SnapshotEventModel | WelcomeEventModel` and `zGetApiLiveResponse` a
`z.union`.

The flattening **drops event names**, so on the client a payload is recognised by shape alone:
`shared/live/stream.ts` tests `'state' in event` and `'operations' in event`; `features/snapshot`
switches on the single-value `state` of each `/api/snapshot` payload. A new event must be listed
in `flattenSSESchema` and stay distinguishable — see "Changing an event touches two places" in
`apps/hump/internal/application/hump_tower/stream_updates/AGENTS.md` and
`apps/hump/internal/application/hump_tower/stream_snapshot/AGENTS.md`.

A payload that fails its schema does not throw out of the iterator: the event is dropped,
`onSseError` is called and the stream ends (`shared/live/stream.integration.test.ts`). All three
stream call sites pass `sseMaxRetryAttempts: 1` and an `onSseError` that logs; `shared/live` and
the `/api/snapshot` query in `features/snapshot` do their own reconnecting.
