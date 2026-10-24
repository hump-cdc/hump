# apps/hump-tower/src/app — how this package hangs together

The FSD **app layer**: the application-wide providers, and nothing else. `index.ts` exports
`Providers`; its one importer is `app/routes/__root.tsx`.

Not to be confused with `apps/hump-tower/app` (no `src`): the rsbuild entry, the router and the
route files. That directory has no guide of its own, so its wiring is described here. It imports
**into** `src`, and nothing in `src` imports it back. `rsbuild.config.ts` and `tsr.config.json`
both hardcode its paths (`app/main.tsx`, `app/routes`, `app/routeTree.gen.ts`).

## Router

`getRouter()` (`app/router.tsx`) sets `defaultNotFoundComponent` to `NotFound` from
`@/pages/not-found`, preloads on `intent` with `defaultPreloadStaleTime: 0`, and turns on scroll
restoration and view transitions.

Route files stay thin: a component, `staticData`, at most `validateSearch` (`operations`). No
route has a `loader` or `beforeLoad`.

## The provider tree

```tsx
<QueryClientProvider client={queryClient}>
    <LiveUpdates />
    <ThemeProvider>
        <TooltipProvider>
            <ConfigProvider>{children}</ConfigProvider>
            <Toaster />
        </TooltipProvider>
    </ThemeProvider>
</QueryClientProvider>
```

What the placement decides:

- **`QueryClientProvider` is outermost** because `LiveUpdates` calls `useQueryClient` and
  `ConfigProvider` calls `useQuery`. The client is a module constant, so it survives re-renders
  and StrictMode remounts. It has no `defaultOptions`; tuning lives in each query hook.
- **`ConfigProvider` gates only `children`**: `ConfigLoading` while pending, `ConfigError` on
  error, the routed tree only after success. Nothing under `<Outlet />` mounts before the
  configuration has loaded.
- **`LiveUpdates` and `Toaster` sit outside that gate.** The live connection opens while the
  configuration is still loading and stays open if it failed; toasts can show in those states too.
- **`ThemeProvider` wraps the routed tree**, so `widgets/sidebar/app-sidebar-theme.tsx` (the only
  `useTheme` caller; `useTheme` throws outside the provider) is always inside it. The `.dark`
  class itself lives on `<html>`, outside React.

`Providers` is mounted by the root **route**, not by `main.tsx`, so `RouterProvider` is above it
and `getRouter()` passes no `context`. The query client is reachable only through React context
below `__root`: a route loader could not use it as things stand.

## What mounting `Providers` does to the backend

- **`GET /api/config`**: `useConfig`, cached forever (`entities/config/AGENTS.md`). Backend:
  `apps/hump/internal/application/hump_tower/get_config/AGENTS.md`.
- **`GET /api/live`**: `LiveUpdates` calls `liveStream.retain()` while mounted. Backend:
  `apps/hump/internal/application/hump_tower/stream_updates/AGENTS.md`; a connected client is what
  turns the collectors on.

`LiveUpdates` only subscribes. **What** gets invalidated is in `shared/state/invalidate.ts`; add a
live-refreshed query there, not here.

`StrictMode` mounts effects twice in dev. `retain` is reference-counted and a released connection
lingers for `LINGER_MS` (1 s, `shared/live/stream.ts`), so the remount reuses the connection
instead of opening a second one.

## What must change together

- **Theme constants.** `shared/theme/theme-script.ts` is injected into `<head>` through
  `html.tags` and sets `.dark` on `<html>` before the bundle runs; `ThemeProvider` toggles the same
  class later. The script inlines the constants from `shared/theme/constants.ts`, but the list of
  theme names is repeated by hand. `styles.css` keys on the same names (`@custom-variant dark`,
  `.dark`, `.theme-transition`).
- **The mount id.** `main.tsx` looks up `#app` and throws without it; the id comes from
  `html.mountId` in `rsbuild.config.ts`, and `styles.css` sizes `#app`.
- **Storybook.** `.storybook/preview.tsx` wraps stories in its own `ThemeProvider` +
  `TooltipProvider` only, not in `Providers`. A provider added here does not reach stories; add it
  to the preview too if a UI-kit component starts to need it.
