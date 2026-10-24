# DropdownMenu

A menu anchored to its trigger, built on `@radix-ui/react-dropdown-menu`: actions, checkbox and
radio items, nested submenus. Pick it for a list of commands or options behind one button; for
a value picker with search use `@/shared/ui/combobox`, for a hint use `@/shared/ui/tooltip`.
Storybook: `Overlays/Dropdown Menu`.

## Parts

Each `DropdownMenu<Part>` wraps the Radix primitive of the same name, forwards its props and sets
`data-slot`. Non-obvious ones:

| Part | Note |
|---|---|
| `DropdownMenu` | Radix `Root` |
| `DropdownMenuContent` | `Portal` + `Content`: always rendered through a portal |
| `DropdownMenuCheckboxItem` | adds a `Check` indicator |
| `DropdownMenuRadioItem` | adds a dot indicator |
| `DropdownMenuShortcut` | a plain `span`: right-aligned shortcut text inside an item |
| `DropdownMenuSubTrigger` | appends a `ChevronRight` |
| `DropdownMenuPortal` | bare portal; `DropdownMenuContent` already has its own |

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

- `DropdownMenuContent`: `sideOffset` defaults to `4` (px between trigger and popup).
- `DropdownMenuItem`: `variant` (`default` / `destructive`, see below) and `inset`.
- `DropdownMenuLabel`, `DropdownMenuSubTrigger`: `inset`.

`inset` only adds `pl-8`, aligning the text with checkbox and radio items, which always carry
`pl-8`: their indicator sits in that gutter at `left-2`. The other parts have no props of their
own.

## Semantic styles

- `DropdownMenuItem` `variant`: `default` — an ordinary action; `destructive` — an irreversible
  action (`text-danger`, highlighted `bg-danger-soft`).
- All highlight styling comes from Radix `data-highlighted` (pointer or keyboard focus); there is
  no separate `hover:` style. An open `DropdownMenuSubTrigger` (`data-state=open`) looks
  highlighted.
- The popups use `shadow-lg` (Tailwind default, not a project token).
- Icons in an item are sized `size-4`; the radio item leaves the size alone because its indicator
  is a `size-2` dot. `DropdownMenuContent` scrolls vertically when it exceeds
  `--radix-dropdown-menu-content-available-height`; `DropdownMenuSubContent` clips instead.
