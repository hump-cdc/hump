# ElapsedTime

An animated duration between two moments: from `startedAt` to `finishedAt`, or to now while
`finishedAt` is not set. Use it for a running or finished process; for a duration you already
have in seconds use `<NumberFlow kind="duration">` from `@/shared/ui/number-flow`.
Storybook: `Content/Elapsed Time`.

## Usage

```tsx
import { ElapsedTime } from '@/shared/ui/elapsed-time'

<span className="font-semibold text-3xl text-ink tracking-tight">
    <ElapsedTime startedAt={startedAt} finishedAt={finishedAt} />
</span>
```

## Props

While `finishedAt` is `undefined` it re-renders once a second from an internal timer, which is
cleared when `finishedAt` arrives or on unmount; with `finishedAt` set the value is static.
`className` goes to `NumberFlow`. Output format is that of `NumberFlow`'s `duration` kind:
`Xh Ym Zs`, leading zero units dropped.

## Semantic styles

None of its own: it renders `<NumberFlow kind="duration">`, which adds `font-mono` and inherits
size, weight and colour from the parent.
