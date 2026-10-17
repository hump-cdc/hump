# ProgressChart

Titled two-tone bar with its percentage, for a rate such as a success rate: `value` percent in
green, the remainder grey or red. Pick `ProgressBar` (`progress-bar`) for plain completion
without a title or a number.
Storybook: `Charts/Progress Chart`.

## Usage

```tsx
import { ProgressChart } from '@/shared/ui/charts/progress-chart'

<ProgressChart title="Success rate" value={stats?.successRate} showError />
```

Pass `undefined` while there is no value: the percentage is hidden and the fill fades out at
its last width, leaving the grey track. The header keeps a fixed height, so nothing shifts.

## Props

Plus the `div` attributes (except `title`) and `ref`.

| Prop | Type | Default | Description |
|---|---|---|---|
| `title` | `ReactNode` | required | label above the bar |
| `value` | `number` | — | share filled green, 0 to 100; the fill width is clamped to that range |
| `showError` | `boolean` | `false` | paints the unfilled remainder red rather than leaving it grey |

## Semantic styles

No cva variants. The colour of the percentage follows `value`:

| `value` | Token |
|---|---|
| below 50 | `text-danger` |
| 50 to below 95 | `text-amber` |
| 95 and above | `text-success` |

- Track `bg-surface-3`; fill `bg-success`; remainder `bg-danger`, visible only with
  `showError` and a value.
- Title `text-ink-subtle`.
- The fill width and both opacities transition over 500ms; the percentage is animated with
  `NumberFlow` (`kind="percent"`).
