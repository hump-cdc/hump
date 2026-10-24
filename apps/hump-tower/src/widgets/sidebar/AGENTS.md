# widgets/sidebar — how this module hangs together

The application shell: the navigation sidebar, the sticky page header, and the slot the current
page renders into. The public contract is `AppLayout`; everything else is module-private. It is
mounted by the pathless layout `app/routes/_sidebar/route.tsx`, so every route under `_sidebar/`
gets the shell. `AppLayout` adds only `SidebarProvider`; the other providers are in `src/app`.

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

`app-sidebar-menu.tsx` is the only place nav entries live. To add a page, follow "Adding a page"
in `apps/hump-tower/AGENTS.md`.

## Header title

The header title does **not** come from the nav label. `AppHeader` reads it through
`useStaticData`, which returns the `staticData` of the deepest matched route that has a `title`.
A route without one renders an empty title; a label and a title that disagree are two separate
strings to fix. `headerAside` travels the same way and is rendered left of `LiveControls`; no route
sets it today.

## Active item

`NavItem` calls `matchRoute({ to, fuzzy: to !== '/' })`: every entry matches its sub-paths too,
except `/`, which must match exactly; fuzzy there would light Flows up on every page.

## Live status and snapshot badge

Both are inside `LiveControls` (`entities/config/state`), not on the nav items, and read the
app-wide `/api/live` stream, so the header shows them on every page. The widget itself imports
nothing from `@/shared/live`.

## Connection info

`ConnectionInfo` reads `useConfig()` and shows `database.host:port`, `name`, `slot` and
`publication`, each with a tooltip. It renders an empty wrapper until the query succeeds.

It is mounted **twice**: in the header inside a `hidden lg:flex` block, and in the sidebar content
inside an `lg:hidden` block under a "Connection" heading. CSS picks the visible one; both share one
cached query.

The version badge is not from the config: `VERSION` is a build-time global (`rsbuild.config.ts`
`source.define`, from `process.env.VERSION`, default `0.0.0`).

## Theme switch

`AppSidebarTheme` toggles between `light` and `dark` only (`THEME_CYCLE`). `system` is a valid
`Theme` and has an icon and label in `THEME_META`, so a stored `system` renders correctly, but the
button can never select it: from `system`, `indexOf` is `-1` and the next theme is `light`.

## Mobile and collapsed

All of it comes from `useSidebar()` (`shared/ui/sidebar`): `isMobile` is `useIsMobile()`, below
1024px, the same edge as the `lg:` classes used here.

- Mobile: `Sidebar` is a sheet. The header's `lg:hidden` `SidebarTrigger` opens it, and `NavItem`
  calls `setOpenMobile(false)` on click; without that the drawer stays over the new page.
- Desktop expanded: the `ChevronsLeft` trigger in the sidebar header collapses it.
- Desktop collapsed: that trigger and the name/version block are hidden
  (`group-data-[collapsible=icon]:hidden`), so `AppHeader` renders its own `ChevronsRight` trigger,
  guarded by `!isMobile && state === 'collapsed'`. Labels hide and `SidebarMenuButton` shows the
  `tooltip` instead, which is why every menu button here passes one.

The collapsed preference and the Cmd/Ctrl+B shortcut belong to `SidebarProvider`.
