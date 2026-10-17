# Placeholder

Inline stand-in for content that is absent, failed or still loading: a round icon badge over a
title and an optional description, centred in a bordered box. For a whole-page error use
`ErrorTerminal` (`error`).
Storybook: `Placeholders/Placeholder`.

Parts (`index.ts`):

- `Placeholder` — the generic one.
- `NoPayloadPlaceholder` — `Placeholder` preset for an operation without a message body
  (`FileX` icon, "No payload", "No message body for the operation.").

## Usage

```tsx
import { NoPayloadPlaceholder, Placeholder } from '@/shared/ui/placeholder'

<Placeholder className="p-12" icon={<Loader2 className="animate-spin" />} title="Loading tables…" />

<Placeholder icon={<TriangleAlert />} title="Failed to load tables" description={errorMessage(error)} />

<NoPayloadPlaceholder />
```

The root is `flex-1 min-h-0`: inside a flex parent it fills the free space and centres its
content.

## Props

`Placeholder` — plus the `div` attributes (except `title`) and `ref`.

| Prop | Type | Default | Description |
|---|---|---|---|
| `icon` | `ReactNode` | — | shown in the round badge; an `svg` inside is sized to `size-4`. No badge without it |
| `title` | `ReactNode` | required | the main line |
| `description` | `ReactNode` | — | the smaller line under the title |

`NoPayloadPlaceholder` — the `Placeholder` props without `icon`, `title` and `description`.

## Semantic styles

No cva variants and no states; the same look serves empty, loading and failed.

- Box: `bg-canvas`, `border-hairline`, `rounded-lg`.
- Icon badge: `bg-surface-2`, `text-ink-tertiary`.
- Title `text-ink-subtle`; description `text-ink-tertiary`.
