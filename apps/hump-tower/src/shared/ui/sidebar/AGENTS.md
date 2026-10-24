# Sidebar

The primitives of the app's navigation sidebar: a fixed column that collapses to an icon rail on
desktop and becomes a drawer below 1024px. These are the building blocks only; the app shell that
fills them (`AppLayout`, nav items, header) is `src/widgets/sidebar`.
Storybook: `Layout/Sidebar`.

## Parts

shadcn's sidebar set: `SidebarProvider` (state, persistence, shortcut, width variables) and
`useSidebar`; `Sidebar`; the blocks `SidebarHeader`, `SidebarContent` (scrollable, takes the free
height), `SidebarFooter`, `SidebarGroup`, `SidebarGroupLabel`, `SidebarGroupContent`,
`SidebarSeparator`; the menu `SidebarMenu` (`ul`), `SidebarMenuItem` (`li`), `SidebarMenuButton`;
`SidebarTrigger`, `SidebarRail` (rendered by `Sidebar` itself) and `SidebarInset` (`main`, the page
area). `constants.ts` exports the widths, `SIDEBAR_STORAGE_KEY` (`sidebar:open`) and
`SIDEBAR_KEYBOARD_SHORTCUT` (`b`).

Every part forwards the props of its HTML element. The exception is `Sidebar` on mobile: it spreads
the rest props onto `Sheet` (a Dialog root that renders no element) and drops `className`, so they
never reach the DOM.

## Usage

```tsx
<SidebarProvider>
    <Sidebar>
        <SidebarHeader>
            <SidebarTrigger className="group-data-[collapsible=icon]:hidden">
                <ChevronsLeft />
            </SidebarTrigger>
        </SidebarHeader>
        <SidebarContent className="px-3">
            <SidebarMenu>
                <SidebarMenuItem>
                    <SidebarMenuButton isActive tooltip="Home">
                        <House />
                        <span>Home</span>
                    </SidebarMenuButton>
                </SidebarMenuItem>
            </SidebarMenu>
        </SidebarContent>
        <SidebarFooter>…</SidebarFooter>
    </Sidebar>
    <SidebarInset>
        <Header>
            <SidebarTrigger className="lg:hidden" />
            …
        </Header>
        …
    </SidebarInset>
</SidebarProvider>
```

- `SidebarProvider` must wrap both `Sidebar` and `SidebarInset`: it is the flex row they sit in.
  `Sidebar`, `SidebarMenuButton`, `SidebarTrigger` and `SidebarRail` call `useSidebar`, which throws
  outside it.
- `SidebarMenuButton` expects an icon followed by a `<span>` label: when collapsed it hides its
  last `span` child and keeps the icon. For a link pass `asChild` and put the icon and the `span`
  inside the child (`widgets/sidebar/nav-item.tsx`).
- React to the collapsed state with `group-data-[collapsible=icon]:…` on anything inside
  `Sidebar`; from code use `useSidebar().state`.
- On mobile nothing closes the drawer after navigation; the caller does it with
  `setOpenMobile(false)` (`widgets/sidebar/nav-item.tsx`, `app-sidebar.tsx`).

## Props

- `SidebarProvider`: `defaultOpen` (default `true`) is the desktop state when nothing is stored;
  `open` / `onOpenChange` make it controlled.
- `Sidebar`: `side` (`'left'` default, or `'right'`); its `div` attributes reach the desktop
  container only.
- `SidebarMenuButton`: `size` (`default` / `sm` / `lg`), `isActive` (sets `data-active`), `asChild`,
  and `tooltip`, a string or `TooltipContent` props.
- `SidebarTrigger`: `children` replaces the default `Menu` icon; `onClick` runs before the toggle.
- `useSidebar().toggleSidebar` toggles the drawer on mobile and `open` on desktop; `isMobile` is
  `useIsMobile()` from `@/shared/utils`.

## Behaviour

- **Desktop** (`lg` and up): `Sidebar` renders a spacer plus a `fixed` full-height container and
  a `SidebarRail`. The root carries `data-state`, `data-side` and `data-collapsible` (`icon` when
  collapsed, empty otherwise); width animates between `--sidebar-width` and
  `--sidebar-width-icon`, both set by `SidebarProvider` and overridable through its `style`.
- **Mobile**: `Sidebar` renders a `Sheet` (`@/shared/ui/sheet`) of width `SIDEBAR_WIDTH_MOBILE`
  driven by `openMobile`, with no close button and a screen-reader-only title.
- **Persistence**: `setOpen` writes `localStorage['sidebar:open']`, also in controlled mode. On
  mount a stored value wins over `defaultOpen`. The drawer state is not stored.
- **Shortcut**: Cmd/Ctrl+B calls `toggleSidebar`.
- **Tooltips**: `SidebarProvider` mounts its own `TooltipProvider` (`delayDuration={0}`).
  `SidebarMenuButton` shows `tooltip` on the right, hidden unless `state === 'collapsed'` and
  not mobile; an object `tooltip` is spread last and can override `side`, `align`, `hidden`.

## Semantic styles

`size` is the only cva variant (`SidebarMenuButton`): `default` `h-9`, `sm` `h-8` with `text-base`,
`lg` `h-11`. Collapsed, every size becomes a `size-9` square with the icon centred.

`SidebarMenuButton` states: rest `text-ink-muted` (icon `text-ink-subtle`); hover `bg-surface-2
text-ink`; active (`data-active=true`) `bg-primary-soft text-primary font-medium`, icon
`text-primary`, unchanged on hover; disabled or `aria-disabled` `opacity-50`.

Surfaces: `Sidebar` `bg-sidebar` with a `border-hairline` on the inner edge; `SidebarInset`
`bg-canvas`. `SidebarRail` is invisible until hovered, then a 1px `bg-hairline-strong` line.
