# shared/ui — UI Kit conventions

Rules for defining components in the UI kit. Follow them for every new component.

## Foundations

- Build on **shadcn/ui + Radix UI** primitives. Pick the Radix primitive that fits the
  component (e.g. `@radix-ui/react-slot`, `@radix-ui/react-dialog`, `@radix-ui/react-dropdown-menu`).
  **If unsure which primitive is appropriate, ask before implementing.** Exceptions: `combobox/`
  is built on Base UI (`@base-ui/react/combobox`), and `where-input/` uses Base UI's popover for
  its suggestion list and `prism-react-editor` for the editor. Other components wrap a library of
  their own: `@tanstack/charts` (`charts/*` except `progress-chart`), `@xyflow/react` (`flow*`),
  `@tanstack/highlight` (`code`), `@number-flow/react`, `sonner` (`toast`),
  `@tanstack/react-table` + `@tanstack/react-virtual` (`data-table`).
- Variants are defined with **`cva`** (`class-variance-authority`).
- Compose class names with **`cn()`** from `@/shared/utils` (clsx + tailwind-merge).
- Prefer standard Tailwind utilities over arbitrary values (`gap-2`, not `gap-[7px]`).
- Use design-token utilities from `app/styles.css` (`bg-primary`, `text-ink-muted`,
  `border-hairline-strong`, …); never raw hex. The one exception is `error/`: the terminal
  window keeps fixed hex colours so it stays dark in both themes (see `error/AGENTS.md`).

## Folder structure

One folder per component; the main file matches the folder.

```
shared/ui/
  <component-name>/
    <component-name>.tsx          // variants + props type + component
    <component-name>.stories.tsx
    index.ts                      // export * from './<component-name>'
```

- `charts/` is a group folder with no `index.ts`, `AGENTS.md` or `CLAUDE.md` of its own; each
  chart lives in a subfolder and is imported as `@/shared/ui/charts/<name>`.
- `index.ts` re-exports with `export * from './<file>'`, one line per public file. A few folders
  use named exports or add `export type` lines; check the folder's `index.ts`.
- Keep `cva` variants, the props `type`, and the component in one `.tsx` file. A larger
  component splits into several files, and each file holds the variants, props and component
  of its own part:
  - compound parts: `<component>-<part>.tsx` (`sheet/sheet-content.tsx`, `tabs/tabs-trigger.tsx`);
  - internal pieces and non-component code: `<component>.<part>.ts(x)`
    (`data-table/data-table.row.tsx`, `flow/flow.helpers.ts`, `*.types.ts`, `*.consts.ts`).
    Exception: every `combobox.<part>` file is public and re-exported from `index.ts`.

  Some existing files deviate (most context files use the hyphen form, `flow*/node.*.tsx`,
  `card/*-card.tsx`, `error/error-terminal.tsx` as the main file); follow the scheme above for
  new files.
- Every component folder carries its own `AGENTS.md` (plus a `CLAUDE.md` holding only
  `@AGENTS.md`) with four parts: what the component is for and its Storybook path, **Usage**,
  **Props**, **Semantic styles**. Do not copy the types into them:
  - **Props** covers only what the types don't make obvious: closed props, where `className`
    and `aria-*` land, controlled/uncontrolled rules, defaults that have a reason.
  - **Semantic styles** covers which variant to use when and token choices that aren't obvious,
    not class lists.

  Keep it in sync when props or variants change.

## Component rules

- Component is a **`React.FC`** with a `type` props alias. Generic components cannot be an
  `FC`: they are generic arrow functions or `function`s (`DataTable`, `Table`). The flow node
  configs (`TableNode`, `SinkNode`, `GhostNode`) are `Object.assign((_props) => null,
  { flowNodeType })`: declarative elements that render nothing.
- **Do not export** the `cva` variants object or the props type — keep them module-private.
  Exceptions: `DataTableProps<T>` (`data-table/data-table.types.ts`) and `SelectableProps`
  (`table/table-row.tsx`).
- Props `type` = element attributes `& VariantProps<typeof xVariants>` (+ extras like
  `asChild`, `ref`).
- Use Radix **`Slot`** for an `asChild` escape hatch when the component renders an element.
- Always pass `className` through `cn(xVariants({ ...variant, className }))`.

### Example (abridged from `button/button.tsx`)

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
  title.

### Groups

Stories are grouped by what the component is for; there is no common root. Pick the group
that fits, and add a new one only when none does — then also add it to `storySort.order`
in `.storybook/preview.tsx`, which fixes the sidebar order of the groups.

| Group | Holds |
|---|---|
| `Actions` | Things the user clicks to do something (`Button`) |
| `Inputs` | Form controls that take a value (`Combobox`, `Text Input`) |
| `Content` | Presentation of data (`Badge`, `Card`, `Data Table`) |
| `Charts` | Charts |
| `Placeholders` | What stands in for content that is absent, failed or still loading (`Error Terminal`, `Progress Bar`) |
| `Overlays` | Layers above the page (`Sheet`, `Tooltip`) |
| `Layout` | Page structure and navigation within it (`Header`, `Sidebar`) |
| `Flows` | The flow diagrams |

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
