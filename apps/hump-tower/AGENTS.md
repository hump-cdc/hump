# apps/hump-tower

The hump web UI: a single-page app over the web UI API (`/api/...` on `WEB_PORT`). Five pages:
Flows (`/`), Configuration, Operations, Snapshot, Statistics. How the Go binary serves the build
(SPA fallback, `routes.json`, 404s) is in `apps/hump/pkg/middleware/AGENTS.md`.

## Stack

- React 19, TypeScript 6 (`strict`), bun as package manager and test runner.
- rsbuild 2 (`rsbuild.config.ts`): entry `app/main.tsx`, output
  `../hump/internal/controller/hump_tower`.
- TanStack Router (file-based), TanStack Query 5, Table 9, Virtual 3; zustand 5 for client state.
- Tailwind v4 through PostCSS; design tokens in `app/styles.css` (base palette → semantic
  light/`.dark` tokens → `@theme` utilities). Radix UI, `@base-ui/react` (the combobox), `cva`,
  `lucide-react`, `@xyflow/react`.
- `@hey-api/openapi-ts` generates the API client, zod 4 schemas and TanStack Query helpers.
- Biome 2 (lint, format, import order). Storybook 10 (`storybook-react-rsbuild`).

## Commands

Scripts are in `package.json`; the moon tasks in `moon.yml` wrap some of them. The ones with a
catch:

| Command | Does |
|---|---|
| `moon run hump-tower:dev` | `bun run dev` (port 3000) with `API_URL=http://localhost:3200`, i.e. against `moon run hump:run` |
| `moon run hump-tower:build` | production build into `apps/hump/internal/controller/hump_tower` |
| `bun run analyze` | build with Rsdoctor; its report goes to `apps/hump/internal/controller/hump_tower/.rsdoctor` (gitignored, not embedded), and the next `build` cleans it |
| `bun run generate:openapi` | `openapi.json` → `src/shared/api/generated`; runs before `dev` and `build` |
| `moon run hump-tower:openapi` | Go `apps/hump/cmd/openapi` → `src/shared/api/openapi.json`; `dev` and `build` depend on it |
| `moon run hump-tower:test-file -- <path>` | `bun test` of one file; the path must start with `/` or `./`, a bare name is a substring filter. `test-cover` / `test-file-cover` add a coverage table in the terminal |
| `moon run hump-tower:typecheck` | `tsc --noEmit`; there is no type-check script in `package.json` |
| `moon run hump-tower:audit` | `bun audit` over the root `bun.lock`, transitive packages included (Dependabot alerts see only direct ones). Local audit checks all packages by default; Actions passes CI exclusions as arguments |
| `moon run hump-tower:registry-check` | checks that every package comes from `registry.npmjs.org`, separately from the vulnerability audit |

`bun test` runs every `*.test.ts(x)`, colocated with the code under test and written on `bun:test`.

There is no dev proxy: `API_URL` and `VERSION` are compile-time constants (`source.define`,
declared in `src/env.d.ts`). An empty `API_URL` means the page's own origin
(`src/shared/api/client.ts`).

## Directory layout

| Path | Holds |
|---|---|
| `app/` | the entry: `main.tsx`, `router.tsx`, `routes/` (route files), `styles.css` |
| `src/app` | the FSD app layer: `Providers` (query client, live updates, theme, tooltips, config, `<Toaster />`) |
| `src/{pages,widgets,features,entities,shared}` | the other FSD layers, see below |
| `.storybook/` | Storybook config; stories come from `src/shared/ui` and `src/widgets` |
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

Principles. Each one is a `noRestrictedImports` override in `biome.json`, per layer:

- **Downward only.** A layer never imports a layer above it.
- **No sideways imports.** A slice does not import another slice of its layer, not even one of
  its own group. The exception is `@x`: the exporting slice adds `@x/<consumer>.ts` listing what
  that consumer may take, e.g. `@/entities/metrics/@x/flows` or
  `@/entities/config/state/@x/operations`.
- **Public API.** A slice is imported through its `index.ts` (`@/pages/flows`,
  `@/entities/statistics/delivered`, `@/shared/state`), never a file inside it. `shared/ui` has no
  barrel of its own: one import per component folder (`@/shared/ui/button`,
  `@/shared/ui/charts/bar-chart`).
- **Relative imports stay inside the slice**: `../../../` is rejected.

A slice is `index.ts` plus `model/` (hooks, stores, pure logic, tests) and `ui/` (components);
either folder is left out when empty; `widgets/sidebar` keeps its files flat. `entities/metrics` is not grouped and has no `index.ts`:
only entities consume it, so its public API is its `@x` files.

Tests that render (`*.test.tsx`) register happy-dom per file and use `@testing-library/react`;
there is no preload, so the other tests run without a DOM (`src/shared/live/AGENTS.md`, "Tests").

## Data flow

- **Requests.** Slices call the generated SDK classes from `@/shared/api` (`Metrics.get`,
  `Operations.list`, ...) inside `useQuery`/`useMutation`, with keys from `keys` in
  `@/shared/state`. `useConfig` is the exception: it spreads the generated `getApiConfigOptions()`.
- **Live updates.** One SSE connection to `/api/live`; `LiveUpdates` in `Providers` turns its
  events into query invalidations. Details in `src/shared/live/AGENTS.md`.
- **Client state** lives in zustand stores inside the owning slice's `model/`.
- **Metrics** arrive already aggregated from `POST /api/metrics`; the UI parses no Prometheus text.

## Generated, never edit by hand

| Path | Produced by |
|---|---|
| `app/routeTree.gen.ts` | the TanStack Router plugin on `dev`/`build`, or `bun run generate-routes` |
| `src/shared/api/openapi.json` | `moon run hump-tower:openapi` |
| `src/shared/api/generated/` | `bun run generate:openapi` |
| `apps/hump/internal/controller/hump_tower/` | `bun run build` |

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
4. `moon run hump-tower:build`: the build writes `routes.json` from the route tree, and the Go
   server answers paths missing from it with status 404.
