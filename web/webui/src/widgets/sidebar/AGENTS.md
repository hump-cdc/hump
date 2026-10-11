# widgets/sidebar — how this module hangs together

The application shell: the navigation sidebar, the sticky page header, and the slot the current
page renders into. FSD layer `widgets`. The public contract is one export, `AppLayout` (`index.ts`);
everything else is module-private. The only consumer is `app/routes/_sidebar/route.tsx`.

## Layout

| File | Role |
|---|---|
| `index.ts` | exports `AppLayout`, nothing else |
| `app-layout.tsx` | `SidebarProvider` → `AppSidebar` + `SidebarInset` (`AppHeader`, then `children`) |
| `app-header.tsx` | page title, sidebar triggers, `ConnectionInfo`, the route's `headerAside`, `LiveControls` |
| `app-sidebar.tsx` | logo + version + collapse trigger, the menu, footer (theme switch, Github, Documentation) |
| `app-sidebar-menu.tsx` | the list of pages, one `NavItem` each — the only place nav entries live |
| `nav-item.tsx` | one menu entry: router `Link`, active state, closes the mobile drawer on click |
| `app-sidebar-theme.tsx` | the light/dark switch |
| `connection-info.tsx` | four badges describing the source database |

```
AppLayout
    AppSidebar
        AppSidebarMenu → NavItem × 5
        ConnectionInfo            (below lg only)
        AppSidebarTheme
    SidebarInset
        AppHeader
            ConnectionInfo        (lg and up only)
            LiveControls
        children                  (the route's <Outlet />)
```

## Where it is mounted

`app/routes/_sidebar/route.tsx` is a pathless layout route that wraps `<Outlet />` in `AppLayout`,
so every file under `app/routes/_sidebar/` gets the shell. The providers it relies on sit one level
up, in `app/routes/__root.tsx` → `@/app` `Providers`: `QueryClientProvider`, `LiveUpdates`,
`ThemeProvider`, `TooltipProvider`, `ConfigProvider`. `AppLayout` adds only `SidebarProvider`.

## Imports

- `@/entities/config/config` — `useConfig`
- `@/entities/config/state` — `LiveControls`
- `@/shared/ui/sidebar`, `@/shared/ui/header`, `@/shared/ui/badge`, `@/shared/ui/tooltip`
- `@/shared/theme` — `useTheme`, `Theme`
- `@/shared/utils` — `useStaticData`

Nothing from `features` or `pages`.

## Adding a page touches three places

1. the page in `src/pages/<name>`;
2. a route file `app/routes/_sidebar/<name>.tsx` with `staticData: { title }`;
3. a `NavItem` in `app-sidebar-menu.tsx` (`to`, `label`, lucide `icon`).

The header title does **not** come from the nav label. `AppHeader` reads it through `useStaticData`,
which returns the `staticData` of the deepest matched route that has a `title`. A route without
one renders an empty title; a label and a title that disagree are two separate strings to fix.
`headerAside` travels the same way and is rendered left of `LiveControls`; no route sets it
today.

## Active item

`NavItem` calls `matchRoute({ to, fuzzy: to !== '/' })`: every entry matches its sub-paths too,
except `/`, which must match exactly — fuzzy there would light Flows up on every page.

## Snapshot badge and live status

There is no badge on the nav items. The snapshot badge lives in the header, inside `LiveControls`
(`entities/config/state`): `SnapshotStatus` reads `useLiveSnapshot()` from `@/shared/live`, renders
only while `state === 'processing'`, shows the percent and links to `/snapshot`. Next to it
`LiveStatus` shows the connection (`useLiveStatus()`) and opens the chart-step dropdown. Both read
the app-wide `/api/live` stream that `LiveUpdates` holds open, so the header shows them on every
page — backend side in `internal/application/webui/stream_updates/AGENTS.md`. The widget itself
imports nothing from `@/shared/live`.

## Connection info

`ConnectionInfo` reads `useConfig()` (`GET /api/config`, cached forever — see
`internal/application/webui/get_config/AGENTS.md`) and shows `database.host:port`, `name`, `slot`
and `publication`, each with a tooltip. It renders an empty wrapper until the query succeeds; in the
app `ConfigProvider` already blocks the tree until then.

It is mounted **twice**: in the header inside a `hidden lg:flex` block, and in the sidebar content
inside an `lg:hidden` block under a "Connection" heading. CSS picks the visible one; both share the
one cached query.

The version badge is not from the config: `VERSION` is a build-time global (`rsbuild.config.ts`
`source.define`, from `process.env.VERSION`, default `0.0.0`; declared in `src/env.d.ts`).

## Theme switch

`AppSidebarTheme` toggles between `light` and `dark` only (`THEME_CYCLE`). `system` is a valid
`Theme` and has an icon and label in `THEME_META`, so a stored `system` renders correctly, but the
button can never select it: from `system`, `indexOf` is `-1` and the next theme is `light`.
Persistence and the `<html>` class are `useTheme().setTheme`'s job (`shared/theme`).

## Mobile and collapsed

All of it comes from `useSidebar()` (`shared/ui/sidebar`): `isMobile` is `useIsMobile()`, below
1024px, the same edge as the `lg:` classes used here.

- Mobile: `Sidebar` is a sheet. The header's `lg:hidden` `SidebarTrigger` opens it, and `NavItem`
  calls `setOpenMobile(false)` on click — without that the drawer stays over the new page.
- Desktop expanded: the trigger with `ChevronsLeft` in the sidebar header collapses it.
- Desktop collapsed: that trigger and the name/version block are hidden
  (`group-data-[collapsible=icon]:hidden`), so `AppHeader` renders its own `ChevronsRight` trigger,
  guarded by `!isMobile && state === 'collapsed'`. Labels hide and `SidebarMenuButton` shows the
  `tooltip` instead, which is why every menu button here passes one.

The collapsed preference and the Cmd/Ctrl+B shortcut belong to `SidebarProvider`.
