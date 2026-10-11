# Sidebar

The primitives of the app's navigation sidebar: a fixed column that collapses to an icon rail on
desktop and becomes a drawer below 1024px. These are the building blocks only; the app shell that
fills them (`AppLayout`, nav items, header) is `src/widgets/sidebar`.
Storybook: `Layout/Sidebar`.

## Parts

| Part | Renders | Role |
|---|---|---|
| `SidebarProvider` | context + `TooltipProvider` + `div` | state, persistence, shortcut, width variables; wraps the whole layout |
| `useSidebar` | hook | reads that state; throws outside `SidebarProvider` |
| `Sidebar` | `div` (desktop) / `Sheet` (mobile) | the sidebar itself |
| `SidebarHeader` | `div` | top block (logo, collapse trigger) |
| `SidebarContent` | `div` | scrollable middle, takes the free height |
| `SidebarFooter` | `div` | bottom block |
| `SidebarGroup` | `div` | a padded section inside the content |
| `SidebarGroupLabel` | `div` | section caption; fades out when collapsed |
| `SidebarGroupContent` | `div` | body of a section |
| `SidebarMenu` | `ul` | list of entries |
| `SidebarMenuItem` | `li` | one entry |
| `SidebarMenuButton` | `button` | the clickable row: icon + label, optional tooltip |
| `SidebarSeparator` | `div` | hairline between blocks |
| `SidebarTrigger` | `button` | toggles the sidebar |
| `SidebarRail` | `button` | thin strip on the sidebar edge that toggles it; `Sidebar` renders it itself |
| `SidebarInset` | `main` | the page area next to the sidebar |

Constants, also exported: `SIDEBAR_WIDTH` (`16rem`), `SIDEBAR_WIDTH_MOBILE` (`18rem`),
`SIDEBAR_WIDTH_ICON` (`4rem`), `SIDEBAR_STORAGE_KEY` (`sidebar:open`), `SIDEBAR_KEYBOARD_SHORTCUT`
(`b`).

Every part forwards the props of its HTML element and sets `data-slot` (most also `data-sidebar`).

## Usage

```tsx
import {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarHeader,
    SidebarInset,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarProvider,
    SidebarSeparator,
    SidebarTrigger,
} from '@/shared/ui/sidebar'

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
  `Sidebar`, `SidebarMenuButton`, `SidebarTrigger` and `SidebarRail` call `useSidebar`.
- `SidebarMenuButton` expects an icon followed by a `<span>` label: when collapsed it hides its
  last `span` child and keeps the icon. For a link pass `asChild` and put the icon and the `span`
  inside the child (`widgets/sidebar/nav-item.tsx`).
- React to the collapsed state with `group-data-[collapsible=icon]:…` on anything inside
  `Sidebar`; from code use `useSidebar().state`.
- On mobile nothing closes the drawer after navigation; the caller does it with
  `setOpenMobile(false)`.

## Props

`SidebarProvider` — `div` attributes plus:

| Prop | Type | Default | Description |
|---|---|---|---|
| `defaultOpen` | `boolean` | `true` | initial desktop state when nothing is stored |
| `open` | `boolean` | — | controlled desktop state |
| `onOpenChange` | `(open: boolean) => void` | — | called instead of the internal setter |

`Sidebar` — `div` attributes plus:

| Prop | Type | Default | Description |
|---|---|---|---|
| `side` | `'left' \| 'right'` | `'left'` | edge the sidebar is attached to (and the drawer slides from) |

`SidebarMenuButton` — `button` attributes plus:

| Prop | Type | Default | Description |
|---|---|---|---|
| `size` | `'default' \| 'sm' \| 'lg'` | `'default'` | row height |
| `isActive` | `boolean` | `false` | marks the current entry (`data-active`) |
| `tooltip` | `string \| TooltipContent props` | — | hint shown only when collapsed on desktop |
| `asChild` | `boolean` | `false` | render the child (e.g. a router `Link`) instead of `button` |

`SidebarGroupLabel` — `div` attributes plus `asChild?: boolean` (default `false`).

`SidebarTrigger` — `button` attributes; `children` replaces the default `Menu` icon, `onClick`
runs before the toggle. The remaining parts have no props of their own.

`useSidebar()` returns:

| Field | Type | Meaning |
|---|---|---|
| `state` | `'expanded' \| 'collapsed'` | desktop state, derived from `open` |
| `open`, `setOpen` | `boolean`, `(open: boolean) => void` | desktop state and its setter |
| `openMobile`, `setOpenMobile` | `boolean`, `(open: boolean) => void` | drawer state and its setter |
| `isMobile` | `boolean` | `useIsMobile()` from `@/shared/utils`: viewport below 1024px |
| `toggleSidebar` | `() => void` | toggles the drawer on mobile, `open` on desktop |

## Behaviour

- **Desktop** (`lg` and up): `Sidebar` renders a spacer plus a `fixed` full-height container and
  a `SidebarRail`. The root carries `data-state`, `data-side` and `data-collapsible` (`icon` when
  collapsed, empty otherwise); width animates between `--sidebar-width` and
  `--sidebar-width-icon`, both set by `SidebarProvider` and overridable through its `style`.
- **Mobile**: `Sidebar` renders a `Sheet` (`@/shared/ui/sheet`) of width `SIDEBAR_WIDTH_MOBILE`
  driven by `openMobile`, with no close button and a screen-reader-only title. `className` is
  applied to the desktop container only.
- **Persistence**: `setOpen` writes `localStorage['sidebar:open']`, also in controlled mode. On
  mount a stored value wins over `defaultOpen`. The drawer state is not stored.
- **Shortcut**: Cmd/Ctrl+B calls `toggleSidebar`.
- **Tooltips**: `SidebarProvider` mounts its own `TooltipProvider` (`delayDuration={0}`).
  `SidebarMenuButton` shows `tooltip` on the right, hidden unless `state === 'collapsed'` and
  not mobile; an object `tooltip` is spread last and can override `side`, `align`, `hidden`.

## Semantic styles

`size` is the only cva variant (`SidebarMenuButton`):

| `size` | Classes |
|---|---|
| `default` | `h-9`, `text-md` |
| `sm` | `h-8`, `text-base` |
| `lg` | `h-11`, `text-md` |

Collapsed, every size becomes a `size-9` square with the icon centred.

`SidebarMenuButton` states:

| State | Tokens |
|---|---|
| rest | `text-ink-muted`; icon `text-ink-subtle` |
| hover, pressed | `bg-surface-2`, `text-ink` |
| active (`data-active=true`) | `bg-primary-soft`, `text-primary`, `font-medium`; icon `text-primary`; hover keeps the same colours |
| focus-visible | `ring-2 ring-primary` |
| disabled / `aria-disabled` | `opacity-50`, `pointer-events-none` |

| Part | Tokens |
|---|---|
| `Sidebar` | `bg-sidebar`, `text-ink`; `border-hairline` on the inner edge (`border-r` for `left`, `border-l` for `right`) |
| `SidebarInset` | `bg-canvas` |
| `SidebarGroupLabel` | `text-ink-subtle`, `text-2xs`, `uppercase`, `tracking-caps`, `font-medium` |
| `SidebarSeparator` | `bg-hairline` |
| `SidebarTrigger` | `text-ink-subtle` → hover `bg-surface-2`, `text-ink`; focus ring `ring-primary` |
| `SidebarRail` | invisible until hover, then a 1px `bg-hairline-strong` line; `cursor-ew-resize`; desktop only |
