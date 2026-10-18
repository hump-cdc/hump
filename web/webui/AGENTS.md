# web/webui

The hump web UI: a single-page app over the web UI API (`/api/...` on `WEB_PORT`). Five pages —
Flows (`/`), Configuration, Operations, Snapshot, Statistics. `bun run build` writes into
`internal/controller/webui`, which is committed and embedded into the Go binary; how it is served
is described in `internal/controller/AGENTS.md`.

## Stack

- React 19, TypeScript 6 (`strict`), bun as package manager and test runner.
- rsbuild 2 (`rsbuild.config.ts`): entry `app/main.tsx`, output `../../internal/controller/webui`.
- TanStack Router (file-based), TanStack Query 5, Table 9, Virtual 3; zustand 5 for client state.
- Tailwind v4 through PostCSS; design tokens in `app/styles.css` (base palette → semantic
  light/`.dark` tokens → `@theme` utilities). Radix UI, `@base-ui/react` (the combobox), `cva`,
  `lucide-react`, `@xyflow/react`.
- `@hey-api/openapi-ts` generates the API client, zod 4 schemas and TanStack Query helpers.
- Biome 2.5.15 (lint, format, import order). Storybook 10 (`storybook-react-rsbuild`).

## Commands

Run in `web/webui`. The `task` targets in the root `taskfile.yaml` wrap some of them.

| Command | Does |
|---|---|
| `bun run dev` | dev server on port 3000; `task web:dev` adds `API_URL=http://localhost:3200` |
| `bun run build` | production build into `internal/controller/webui`; `task web:build` |
| `bun run preview` | `rsbuild preview`: serves the last production build locally |
| `bun run analyze` | production build with Rsdoctor (`RSDOCTOR=true`): opens the bundle analysis page; its report goes to `internal/controller/webui/.rsdoctor` (gitignored, not embedded) and the next `build` cleans it |
| `bun run check` / `lint` / `format` | `biome check` / `biome lint` / `biome format` |
| `bun test` | every `*.test.ts(x)`, colocated with the code under test, written on `bun:test` |
| `bun run storybook` | Storybook on port 6006; `task web:storybook` |
| `bun run build-storybook` | `storybook build` → static Storybook in `storybook-static/` (gitignored) |
| `bun run knip` | knip (`knip.jsonc`): unused files, exports, types and dependencies; must exit 0 |
| `bun run generate:openapi` | `openapi.json` → `src/shared/api/generated`; runs before `dev`, `build` |
| `bun run generate-routes` | `tsr generate` → `app/routeTree.gen.ts` |
| `task openapi` | Go `cmd/openapi` → `src/shared/api/openapi.json`; `web:dev`, `web:build` depend on it |

There is no dev proxy: `API_URL` and `VERSION` are compile-time constants (`source.define`,
declared in `src/env.d.ts`). An empty `API_URL` means the page's own origin
(`src/shared/api/client.ts`). There is no type-check script (`tsconfig.json` has `noEmit`);
`task check` runs `bun run check`, `bun test` and `bunx tsc --noEmit` here, after the Go lint and tests.

## Directory layout

| Path | Holds |
|---|---|
| `app/` | the entry: `main.tsx`, `router.tsx`, `routes/` (route files), `styles.css` |
| `src/app` | the FSD app layer: `Providers` (query client, live updates, theme, tooltips, config, `<Toaster />`) |
| `src/{pages,widgets,features,entities,shared}` | the other FSD layers, see below |
| `.storybook/` | Storybook config; picks up `src/shared/ui` (stories, `.mdx`); the `src/widgets` stories glob is configured, but there are no widget stories yet |
| `public/` | fonts, logos, favicon |

`@/*` resolves to `src/*` (`tsconfig.json`). `app/` is reached by relative paths only.

## Architecture

Feature-Sliced Design. Imports point downwards only.

| Layer | Role | May import |
|---|---|---|
| `app/`, `src/app` | entry, router, route files, providers | every layer below |
| `src/pages` | one slice per page, composing lower layers | `widgets`, `features`, `entities`, `shared` |
| `src/widgets` | the app shell (`sidebar`) | `features`, `entities`, `shared` |
| `src/features` | user scenarios (`snapshot`) | `entities`, `shared` |
| `src/entities` | domain slices, mostly grouped: `entities/<group>/<slice>` | `shared` |
| `src/shared` | API client, live stream, state keys, theme, UI kit, utils | `shared` only |

Principles — each one is a `noRestrictedImports` override in `biome.json`, per layer:

- **Downward only.** A layer never imports a layer above it.
- **No sideways imports.** A slice does not import another slice of its layer, not even one of
  its own group. The exception is `@x`: the exporting slice adds `@x/<consumer>.ts` listing what
  that consumer may take, e.g. `@/entities/metrics/@x/flows` or
  `@/entities/config/state/@x/operations`. Only `entities` uses it today.
- **Public API.** A slice is imported through its `index.ts` (`@/pages/flows`,
  `@/entities/statistics/delivered`, `@/shared/state`), never a file inside it. `shared/ui` has no
  barrel of its own: one import per component folder (`@/shared/ui/button`,
  `@/shared/ui/charts/bar-chart`).
- **Relative imports stay inside the slice**: `../../../` is rejected.

`bun run lint` passes on the current tree, so none of these is violated anywhere.

A slice is `index.ts` plus `model/` (hooks, stores, pure logic, tests) and `ui/` (components).
`widgets/sidebar` keeps its files flat. Every page except `pages/operations` has no `model/`,
and neither do `entities/config/source-viewer` and `entities/flows/toolbar`. `entities/metrics`
is not grouped, has no `ui/` and no `index.ts`: only entities consume it, so its public API is
its `@x` files.

Tests that render (`*.test.tsx`) register happy-dom per file and use `@testing-library/react`;
there is no preload, so the other tests run without a DOM (`src/shared/live/AGENTS.md`, "Tests").

## Data flow

- **Requests.** Slices call the generated SDK classes from `@/shared/api` (`Metrics.get`,
  `Operations.list`, ...) inside `useQuery`/`useMutation`, with keys from `keys` in
  `@/shared/state`. `useConfig` is the exception: it spreads the generated `getApiConfigOptions()`.
- **Live updates.** `shared/live` holds one SSE connection to `/api/live` (`liveStream`), which
  reconnects with backoff. `LiveUpdates`, mounted in `Providers`, invalidates the metrics queries
  on every batch, and metrics plus operations after a reconnect (`shared/state/invalidate.ts`).
  Slices subscribe themselves with `useLiveOperations`, `useLiveStatus`, `useLiveSnapshot`.
- **Client state** lives in zustand stores inside the owning slice's `model/`.
- **Metrics** arrive already aggregated from `POST /api/metrics`; the UI parses no Prometheus text.

## Generated — never edit by hand

| Path | Produced by |
|---|---|
| `app/routeTree.gen.ts` | the TanStack Router plugin on `dev`/`build`, or `generate-routes` |
| `src/shared/api/openapi.json` | `task openapi` |
| `src/shared/api/generated/` | `bun run generate:openapi` |
| `internal/controller/webui/` | `bun run build` |

All four are committed. Biome skips the first three, and `app/styles.css`.

## Conventions

- Formatting (Biome): 4 spaces, 120 columns, single quotes, no semicolons, trailing commas.
- Import groups, blank line between: `@tanstack/*`; `react`; bun/node; packages; `@/` layers;
  relative paths. Biome also wants Tailwind classes sorted, inside `cn`, `cva` and `clsx` too.
- Files and folders are kebab-case; parts of one component share a dotted prefix
  (`operation-details.tabs.tsx`). Components are named exports typed as `FC`.

## Adding a page

1. Page slice `src/pages/<name>/`: `ui/<name>.tsx` exporting the component, re-exported from
   `index.ts`.
2. Route file `app/routes/_sidebar/<name>.tsx`. `_sidebar` is the pathless layout that wraps the
   page in `AppLayout`; `staticData.title` is what the header shows.

    ```tsx
    export const Route = createFileRoute('/_sidebar/statistics')({
        component: Statistics,
        staticData: { title: 'Statistics' },
    })
    ```

3. A `<NavItem to label icon />` in `src/widgets/sidebar/app-sidebar-menu.tsx`.
4. `task web:build`: the build writes `routes.json` from the route tree, and the Go server answers
   paths missing from it with status 404.

## Module guides

Each module carries its own `AGENTS.md` with the details: `src/app`,
`src/entities/{config,flows,metrics,operations,statistics}`, `src/features/snapshot`,
`src/pages/{configuration,flows,not-found,operations,snapshot,statistics}`,
`src/shared/{api,live,state,theme,ui,utils}`, `src/widgets/sidebar`.
