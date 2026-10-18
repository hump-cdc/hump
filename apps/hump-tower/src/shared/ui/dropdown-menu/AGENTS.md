# DropdownMenu

A menu anchored to its trigger, built on `@radix-ui/react-dropdown-menu`: actions, checkbox and
radio items, nested submenus. Pick it for a list of commands or options behind one button; for
a value picker with search use `@/shared/ui/combobox`, for a hint use `@/shared/ui/tooltip`.
Storybook: `Overlays/Dropdown Menu`.

## Parts

| Part | Renders | Role |
|---|---|---|
| `DropdownMenu` | `Root` | state holder: `open` / `onOpenChange` / `defaultOpen` / `modal` |
| `DropdownMenuTrigger` | `Trigger` | opens the menu; `asChild` to wrap a `Button` |
| `DropdownMenuContent` | `Portal` + `Content` | the popup, always rendered through a portal |
| `DropdownMenuGroup` | `Group` | groups related items |
| `DropdownMenuLabel` | `Label` | non-interactive heading |
| `DropdownMenuItem` | `Item` | one action (`onSelect`, `disabled`) |
| `DropdownMenuCheckboxItem` | `CheckboxItem` | toggle (`checked`, `onCheckedChange`) with a `Check` indicator |
| `DropdownMenuRadioGroup` | `RadioGroup` | single choice (`value`, `onValueChange`) |
| `DropdownMenuRadioItem` | `RadioItem` | one option (`value`) with a dot indicator |
| `DropdownMenuSeparator` | `Separator` | hairline between sections |
| `DropdownMenuShortcut` | `span` | right-aligned shortcut text inside an item |
| `DropdownMenuSub` | `Sub` | submenu state |
| `DropdownMenuSubTrigger` | `SubTrigger` | item that opens the submenu; appends a `ChevronRight` |
| `DropdownMenuSubContent` | `SubContent` | the submenu popup |
| `DropdownMenuPortal` | `Portal` | bare portal; `DropdownMenuContent` already has its own |

Every part forwards the props of its Radix primitive (or of `span`) and sets `data-slot`.

## Usage

```tsx
import {
    DropdownMenu,
    DropdownMenuCheckboxItem,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuRadioGroup,
    DropdownMenuRadioItem,
    DropdownMenuSeparator,
    DropdownMenuShortcut,
    DropdownMenuSub,
    DropdownMenuSubContent,
    DropdownMenuSubTrigger,
    DropdownMenuTrigger,
} from '@/shared/ui/dropdown-menu'

<DropdownMenu>
    <DropdownMenuTrigger asChild>
        <Button variant="outline">Open menu</Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="start" className="w-56">
        <DropdownMenuLabel>My Account</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={onProfile}>
            <User />
            Profile
            <DropdownMenuShortcut>⇧⌘P</DropdownMenuShortcut>
        </DropdownMenuItem>
        <DropdownMenuCheckboxItem checked={showStatus} onCheckedChange={setShowStatus}>
            Show status bar
        </DropdownMenuCheckboxItem>
        <DropdownMenuLabel inset>Panel position</DropdownMenuLabel>
        <DropdownMenuRadioGroup value={position} onValueChange={setPosition}>
            <DropdownMenuRadioItem value="top">Top</DropdownMenuRadioItem>
            <DropdownMenuRadioItem value="bottom">Bottom</DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
        <DropdownMenuSub>
            <DropdownMenuSubTrigger>More options</DropdownMenuSubTrigger>
            <DropdownMenuSubContent>
                <DropdownMenuItem onSelect={onInvite}>Invite users</DropdownMenuItem>
            </DropdownMenuSubContent>
        </DropdownMenuSub>
        <DropdownMenuSeparator />
        <DropdownMenuItem variant="destructive" onSelect={onLogout}>
            <LogOut />
            Log out
        </DropdownMenuItem>
    </DropdownMenuContent>
</DropdownMenu>
```

The only app call site, `entities/config/state/ui/live-controls.tsx`, uses `DropdownMenuContent`
as a plain popover with arbitrary children and no items; the story is the reference for the
item parts.

## Props

`DropdownMenuContent` — Radix `Content` props (`align`, `side`, …), with one default changed:

| Prop | Type | Default | Description |
|---|---|---|---|
| `sideOffset` | `number` | `4` | gap between trigger and popup, px |

`DropdownMenuItem` — Radix `Item` props plus:

| Prop | Type | Default | Description |
|---|---|---|---|
| `variant` | `'default' \| 'destructive'` | `'default'` | tone of the action |
| `inset` | `boolean` | `false` | left padding `pl-8`, aligning the text with checkbox / radio items |

`DropdownMenuLabel`, `DropdownMenuSubTrigger` — Radix `Label` / `SubTrigger` props plus:

| Prop | Type | Default | Description |
|---|---|---|---|
| `inset` | `boolean` | `false` | left padding `pl-8`, same alignment as above |

The other parts have no props of their own.

## Semantic styles

cva variants of `DropdownMenuItem`:

| `variant` | Tokens | Meaning |
|---|---|---|
| `default` | `text-ink-muted`; highlighted `bg-surface-1`, `text-ink` | ordinary action |
| `destructive` | `text-danger`; highlighted `bg-danger-soft`, `text-danger` | irreversible action |

`inset` (`DropdownMenuItem`, `DropdownMenuLabel`, `DropdownMenuSubTrigger`) only adds `pl-8`.
Checkbox and radio items always carry `pl-8`: the indicator sits in that gutter at `left-2`.

| Part | Tokens |
|---|---|
| `DropdownMenuContent`, `DropdownMenuSubContent` | `bg-canvas`, `text-ink`, `border-hairline`, `rounded-lg`; `shadow-lg` (Tailwind default, not a project token) |
| checkbox / radio item, sub trigger | `text-ink-muted`; highlighted `bg-surface-1`, `text-ink` |
| `DropdownMenuLabel` | `text-ink-subtle`, `font-medium` |
| `DropdownMenuShortcut` | `text-ink-subtle`, `text-xs`, `ml-auto` |
| `DropdownMenuSeparator` | `bg-hairline` |

States, all from Radix data attributes:

- `data-highlighted` (pointer or keyboard focus) — the highlight colours above; there is no
  separate `hover:` style.
- `data-disabled` — `opacity-50`, `pointer-events-none`.
- `data-state=open` on `DropdownMenuSubTrigger` — same colours as highlighted.
- `data-state` / `data-side` on the popups — fade, zoom and a 1-unit slide away from the trigger.

Icons placed in an item are sized to `size-4` (`[&_svg]:size-4`); the radio item leaves the size
alone because its indicator is a `size-2` dot. `DropdownMenuContent` scrolls vertically when it
exceeds `--radix-dropdown-menu-content-available-height`; `DropdownMenuSubContent` clips instead.
