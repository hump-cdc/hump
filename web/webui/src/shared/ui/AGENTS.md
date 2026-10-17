# shared/ui — UI Kit conventions

Rules for defining components in the UI kit. Follow them for every new component.

## Foundations

- Build on **shadcn/ui + Radix UI** primitives. Pick the Radix primitive that fits the
  component (e.g. `@radix-ui/react-slot`, `@radix-ui/react-dialog`, `@radix-ui/react-dropdown-menu`).
  **If unsure which primitive is appropriate, ask before implementing.** Exceptions: `combobox/`
  is built on Base UI (`@base-ui/react/combobox`), `where-input/` uses Base UI's popover
  (`@base-ui/react/popover`) for its suggestion list and `prism-react-editor` for the editor, and
  `charts/area-chart`, `charts/bar-chart` and `charts/chart` are built on `@tanstack/charts`
  (`charts/progress-chart` is not). Other components wrap a library of their own: `flow`,
  `flow-detailed` and `flow-summary` use `@xyflow/react`, `code` uses `@tanstack/highlight`,
  `number-flow` uses `@number-flow/react`, `toast` uses `sonner`, and `data-table` uses
  `@tanstack/react-table` with `@tanstack/react-virtual`.
- Variants are defined with **`cva`** (`class-variance-authority`).
- Compose class names with **`cn()`** from `@/shared/utils` (clsx + tailwind-merge).
- Prefer standard Tailwind utilities over arbitrary values (`gap-2`, not `gap-[7px]`).
- Use design-token utilities from `app/styles.css` (`bg-primary`, `text-ink-muted`,
  `border-hairline-strong`, …); never raw hex. The one exception is `error/`: the terminal
  window keeps fixed hex colours so it stays dark in both themes (see `error/AGENTS.md`).

## Folder structure

One folder per component, kebab-case name; the file matches the folder. No
`shared/ui/index.ts` barrel — import each component from its own folder.

```
shared/ui/
  <component-name>/
    <component-name>.tsx          // variants + props type + component
    <component-name>.stories.tsx
    index.ts                      // export * from './<component-name>'
```

- Folder/file: **kebab-case** (`filter-dropdown/filter-dropdown.tsx`).
- Exceptions to the layout above: `charts/` is a group folder with no `index.ts` of its own;
  each chart lives in a subfolder (`area-chart`, `bar-chart`, `chart`, `progress-chart`) and is
  imported as `@/shared/ui/charts/<name>`. `charts/chart/` has no `chart.tsx`, only `chart-*`
  files. `error/` has no `error.tsx`: its main file is `error-terminal.tsx`.
- `index.ts` re-exports with `export * from './<file>'`, one line per component or part file.
  Exceptions: `elapsed-time`, `mark`, `toast`, `where-input` and `charts/chart` use named
  exports; `multi-combo-box` adds `export type * from './multi-combo-box.types'`; `table` also
  re-exports `./table.types` and adds `export type { TableVariant } from './table-context'`;
  `data-table` also re-exports `./data-table.types`; `combobox` also re-exports
  `./combobox.helpers`; `sidebar` also re-exports `./constants`.
- Exported component: **PascalCase** (`FilterDropdown`).
- Keep `cva` variants, the props `type`, and the component in one `.tsx` file. A larger
  component splits into several files, and each file holds the variants, props and component
  of its own part:
  - compound parts: `<component>-<part>.tsx` (`sheet/sheet-content.tsx`,
    `tabs/tabs-trigger.tsx`; also `card`, `dropdown-menu`, `sidebar`, `table`, `tooltip`, `header`, `error`
    (`error-line.tsx`, `error-prompt.tsx`) and `charts/chart` (`chart-*`));
  - internal pieces and non-component code: `<component>.<part>.ts(x)`
    (`data-table/data-table.row.tsx`, `flow/flow.helpers.ts`, `*.types.ts`, `*.consts.ts`).
    `combobox` is the exception: all its `combobox.<part>` files (`combobox.item.tsx` etc., plus
    `combobox.helpers.ts`) are public, re-exported from `index.ts`. Most context files use the
    hyphen form instead (`card/card-context.ts`, `table/table-context.ts`,
    `sidebar/sidebar-context.tsx`, `charts/chart/chart-context.ts`), and so does the internal
    `tabs/tabs-list-indicator.tsx`; `flow-detailed/flow-detailed.context.ts` keeps the dot form;
  - the remaining files use other names: `card/{area-chart,bar-chart,info,metric}-card.tsx`,
    `flow/{metric-hint,node.label,node.table,sink-icon}.tsx`, `flow/node.ts`,
    `flow-detailed/node.{ghost,sink,sink.target}.tsx`, `flow-summary/node.sink.tsx`,
    `number-flow/duration-flow.tsx`, `placeholder/no-payload-placeholder.tsx`,
    `sidebar/constants.ts`.

  Only `data-table` keeps its component props type outside the `.tsx`: `DataTableProps<T>` is in
  `data-table.types.ts`.
- Every component folder carries its own `AGENTS.md` (plus a `CLAUDE.md` holding only
  `@AGENTS.md`; the `charts/` group folder has neither, its subfolders do) with four parts: what the component is for and its Storybook path, **Usage**,
  **Props**, **Semantic styles** (variants and the design tokens behind them). Keep it in
  sync when props or variants change.

## Component rules

- Component is a **`React.FC`** with a `type` props alias. Generic components cannot be an
  `FC`: `DataTable` is a generic arrow function returning `ReactElement` (its internal
  `DataTableHeader`, `HeaderCell` and `ResizeGrip` are generic arrows returning `ReactNode`;
  `DataTableRow` is a `memo` of a generic arrow, cast to `<T>(props) => ReactNode`),
  `Table`, `TableRow` and `TableRowHeader` generic `function`s returning `ReactNode`. The flow
  node configs `TableNode` (`flow/node.table.tsx`), `SinkNode` (`flow-detailed/node.sink.tsx`,
  `flow-summary/node.sink.tsx`) and `GhostNode` (`flow-detailed/node.ghost.tsx`) are
  `Object.assign((_props) => null, { flowNodeType })`: declarative elements that render nothing.
  `Combobox` and `ComboboxValue` are not components of their own: they re-export Base UI's
  `Combobox.Root` and `Combobox.Value`.
- **Do not export** the `cva` variants object or the props type — keep them module-private.
  Exceptions: `DataTableProps<T>` (`data-table/data-table.types.ts`, re-exported from
  `index.ts`) and `SelectableProps` (`table/table-row.tsx`) are exported.
- Props `type` = element attributes `& VariantProps<typeof xVariants>` (+ extras like
  `asChild`, `ref`).
- Use Radix **`Slot`** for an `asChild` escape hatch when the component renders an element.
- Always pass `className` through `cn(xVariants({ ...variant, className }))`.

### Example (`button/button.tsx`)

```tsx
import type { ButtonHTMLAttributes, FC, Ref } from 'react'

import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/shared/utils'

const buttonVariants = cva('inline-flex items-center justify-center rounded-lg ...', {
    variants: {
        variant: {
            primary: 'bg-primary text-on-primary hover:bg-primary-hover',
            // …
        },
    },
    defaultVariants: {
        variant: 'primary',
    },
})

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> &
    VariantProps<typeof buttonVariants> & {
        asChild?: boolean
        ref?: Ref<HTMLButtonElement>
    }

export const Button: FC<ButtonProps> = ({ className, variant, asChild = false, ...props }) => {
    const Comp = asChild ? Slot : 'button'

    return <Comp className={cn(buttonVariants({ variant, className }))} {...props} />
}
```

`index.ts`:

```ts
export * from './button'
```

## Storybook

Every component ships a story, except `flow/` and `charts/chart/`: neither renders on its own
(see their `AGENTS.md`). Keep **one story per component** whenever possible — if
you think a component needs more than one, **ask first**.

Infer the story shape from `button/button.stories.tsx`:

- `StoryArgs` reuses the component's props via `Pick<ComponentProps<typeof X>, …>` for
  real props, plus story-only fields (e.g. `label`, `icon`) when needed.
- `meta.title` = `'<Group>/<Component>'` (see [Groups](#groups)); drive everything through
  `argTypes` + `args` and a single `render`.
- Export one story `Default` whose `name` repeats the last segment of `meta.title`
  (`'Actions/Button'` → `'Button'`), so a single-story component shows no extra nesting under its
  title. Four stories use the PascalCase component name instead: `CopyButton`, `DropdownMenu`,
  `OperationBadge`, `OperationStatusBadge`. `placeholder/placeholder.stories.tsx` also exports
  `NoPayload` (`'No Payload'`) for `NoPayloadPlaceholder`.

### Groups

Stories are grouped by what the component is for; there is no common root. Pick the group
that fits, and add a new one only when none does — then also add it to `storySort.order`
in `.storybook/preview.tsx`, which fixes the sidebar order of the groups.

| Group | Holds |
|---|---|
| `Actions` | Things the user clicks to do something: `Button`, `Copy Button` |
| `Inputs` | Form controls that take a value: `Checkbox`, `Combobox`, `Multi Combo Box`, `Text Input`, `Where Input` |
| `Content` | Presentation of data: `Badge`, `Card` (plus `Card/Area`, `Card/Bar`, `Card/Info`, `Card/Metric`), `Code`, `Data Table`, `Elapsed Time`, `Mark`, `Number Flow`, `Operation Badge`, `Operation Status Badge`, `Sink`, `Table` |
| `Charts` | Charts: `Area Chart`, `Bar Chart`, `Progress Chart` |
| `Placeholders` | What stands in for content that is absent, failed or still loading: `Error Terminal`, `Placeholder`, `Progress Bar` |
| `Overlays` | Layers above the page: `Dropdown Menu`, `Sheet`, `Toast`, `Tooltip` |
| `Layout` | Page structure and navigation within it: `Header`, `Sidebar`, `Tabs`, `Toolbar` |
| `Flows` | The flow diagrams: `Flow Detailed`, `Flow Summary` |

`Colors` (`colors.mdx`) sits at the top level, outside any group. `storySort.order` also lists
`Widgets` after `Flows`; no story uses it yet.

### Naming & grouping controls

- Give every `argType` a human-readable `name` (the control's label in the Controls
  panel) — don't rely on the raw prop name.
- Group related controls with `table.category`. Use a small, consistent set of
  categories: **Content**, **Style**, **State** (add more only when needed).
- When a control's label reads opposite to the prop (e.g. `nonInteractive` shown as
  `Interactive`), keep in mind the boolean value is inverted — call it out if confusing.

### Event handlers → Actions pane

- Expose each event handler (`onClick`, `onChange`, …) as a story arg typed from React
  (e.g. `MouseEventHandler<HTMLButtonElement>`) and pass it through to the component.
- Default it to `fn()` from `storybook/test` in `args`. The actions addon auto-logs
  spied `fn()` callbacks, so interactions show up in the **Actions** pane.
- Hide handlers from the Controls panel with `table: { disable: true }` — they have no
  meaningful input control, but still log to the **Actions** pane.

```tsx
import type { ComponentProps, MouseEventHandler } from 'react'

import type { Meta, StoryObj } from '@storybook/react'
import { fn } from 'storybook/test'

import { Button } from './button'

type StoryArgs = Pick<ComponentProps<typeof Button>, 'variant' | 'disabled'> & {
    label: string
    onClick: MouseEventHandler<HTMLButtonElement>
}

const meta: Meta<StoryArgs> = {
    title: 'Actions/Button',
    argTypes: {
        label: { name: 'Content', control: 'text', table: { category: 'Content' } },
        variant: {
            name: 'Variant',
            control: 'select',
            options: ['primary', 'secondary', 'outline', 'ghost', 'danger'],
            table: { category: 'Style' },
        },
        disabled: { name: 'Disabled', control: 'boolean', table: { category: 'State' } },
        onClick: { table: { disable: true } },
    },
    args: { variant: 'primary', label: 'Button', disabled: false, onClick: fn() },
    render: ({ variant, label, disabled, onClick }) => (
        <Button variant={variant} disabled={disabled} onClick={onClick}>
            {label}
        </Button>
    ),
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Button',
}
```

The Storybook preview renders light/dark side-by-side automatically — no per-theme story.
