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

`value` is 0 to 100; the fill width is clamped to that range. `showError` paints the unfilled
remainder red instead of leaving it grey. `title` replaces the native `div` attribute.

## Semantic styles

No cva variants. The percentage colour follows `value`: below 50 `text-danger`, 50 to below 95
`text-amber`, 95 and above `text-success`. The remainder is `bg-danger` only with `showError`
and a value; otherwise the `bg-surface-3` track shows through.
