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
embed in a card or a panel.

## Props

`ErrorTerminal` takes no `className`, element attributes or `ref`. Its children are the terminal
body; `status` is rendered after them as `status: <status>`, before the cursor. `title` is the
`h1` under the window, `terminalTitle` the text in its title bar.

## Semantic styles

The terminal window is dark in both themes, so its colours are fixed hex values in the source;
only the page around it follows the theme.

`ErrorLine` `tone`: `default` — plain output; `muted` — secondary output; `danger` — an error;
`warning` — a warning.
