# Header

The sticky bar at the top of the page area: a title on the left, actions on the right. One per
page shell; the app renders it once, in `widgets/sidebar/app-header.tsx`. For a row of filters
inside a page use `@/shared/ui/toolbar`.
Storybook: `Layout/Header`.

## Parts

| Part | Renders | Role |
|---|---|---|
| `Header` | `header` | the bar; lays its children out with `justify-between` |
| `HeaderTitle` | `h1` | the page title, truncated when it does not fit |

Both forward the props of their HTML element and set `data-slot`.

## Usage

```tsx
import { Header, HeaderTitle } from '@/shared/ui/header'

<Header>
    <div className="flex min-w-0 items-center gap-2.5">
        <HeaderTitle>Overview</HeaderTitle>
        <Badge variant="active" border dot>
            production
        </Badge>
    </div>
    <Button variant="outline">
        <RefreshCw />
        Refresh
    </Button>
</Header>
```

`Header` spreads its direct children to the two ends, so group the left side (trigger, title,
badges) in one element. That wrapper needs `min-w-0` for `HeaderTitle` to truncate.

## Props

Neither part has props of its own: `Header` takes `header` attributes, `HeaderTitle` takes `h1`
attributes.

## Semantic styles

No cva variants.

| Part | Classes | Meaning |
|---|---|---|
| `Header` | `bg-sidebar`, `border-hairline border-b` | same surface as the sidebar, separated from the page by a hairline |
| `Header` | `sticky top-0 z-10`, `h-14`, `px-4 lg:px-6` | stays on top of the scrolling page; fixed height |
| `HeaderTitle` | `text-ink`, `text-lg`, `font-semibold`, `tracking-heading` | heading scale (15px, -0.3px tracking) |

No hover, active or disabled states.
