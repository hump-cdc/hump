# ErrorTerminal

Full-page error screen: a fake terminal window with a short "session", then a headline, a
description and an optional action. Used for the 404 page and for a failed configuration load.
For an error inside a page use `Placeholder` (`placeholder`).
Storybook: `Placeholders/Error Terminal`.

Parts (`index.ts`):

- `ErrorTerminal` — the page: the terminal window, the status line with a blinking cursor,
  the headline block.
- `ErrorPrompt` — a command line of the session: `➜ ~/hump <children>`. The `~/hump` path is
  fixed.
- `ErrorLine` — an output line of the session, coloured by `tone`.

## Usage

```tsx
import { ErrorLine, ErrorPrompt, ErrorTerminal } from '@/shared/ui/error'

<ErrorTerminal
    status="404 — page not found"
    title="We grepped. We found nothing."
    description="Error 404: this page has left the cluster."
    action={
        <Button asChild>
            <Link to="/" replace>
                Go to Flows
            </Link>
        </Button>
    }
>
    <ErrorPrompt>cd {path}</ErrorPrompt>
    <ErrorLine tone="danger">cd: no such file or directory: {path}</ErrorLine>

    <ErrorPrompt>git blame</ErrorPrompt>
    <ErrorLine tone="warning">fatal: it was DNS. it&apos;s always DNS.</ErrorLine>
</ErrorTerminal>
```

`ErrorTerminal` is `min-h-screen` and centres itself: it is the whole page, not something to
embed in a card or a panel. The children are the terminal body; `status: <status>` is appended
after them.

## Props

`ErrorTerminal` — these props only: no `className`, no element attributes, no `ref`.

| Prop | Type | Default | Description |
|---|---|---|---|
| `status` | `ReactNode` | — | last terminal line, rendered as `status: <status>` before the cursor |
| `title` | `ReactNode` | — | the `h1` under the terminal |
| `description` | `ReactNode` | — | the paragraph under the title |
| `action` | `ReactNode` | — | slot under the description, usually a `Button` |
| `terminalTitle` | `ReactNode` | `'bash — hump — 80×24'` | text in the window title bar |
| `children` | `ReactNode` | — | the session: `ErrorPrompt` and `ErrorLine` rows |

`ErrorPrompt` — `children` only (the command text).

`ErrorLine` — plus the `div` attributes.

| Prop | Type | Default | Description |
|---|---|---|---|
| `tone` | `'default' \| 'muted' \| 'danger' \| 'warning'` | `'default'` | colour of the line |

## Semantic styles

The terminal window is dark in both themes: its colours are fixed hex values in the source,
not design tokens. Only the page around it follows the theme.

`ErrorLine` cva `tone`:

| `tone` | Colour | Meaning |
|---|---|---|
| `default` | `#f7f8f8` | plain output |
| `muted` | `#8a8f98` | secondary output |
| `danger` | `#eb5757` | an error |
| `warning` | `#f2c94c` | a warning |

- `ErrorPrompt`: arrow `#27a644`, path `#5e6ad2`, command `#f7f8f8`.
- Terminal: window `#0f1011` with a `#23252a` border, title bar `#141516`, body `bg-black`,
  status line `#d0d6e0`, traffic lights `#eb5757` / `#f2c94c` / `#27a644`.
- Tokens: page `bg-canvas` with a dot grid in `--hairline-strong`; title `text-ink`;
  description `text-ink-subtle`; cursor `bg-primary` with `animate-blink`.
