# shared/ui — UI Kit conventions

Rules for defining components in the UI kit. Follow them for every new component.

## Foundations

- Build on **shadcn/ui + Radix UI** primitives. Pick the Radix primitive that fits the
  component (e.g. `@radix-ui/react-slot`, `@radix-ui/react-dialog`, `@radix-ui/react-dropdown-menu`).
  **If unsure which primitive is appropriate, ask before implementing.**
- Variants are defined with **`cva`** (`class-variance-authority`).
- Compose class names with **`cn()`** from `@/shared/utils` (clsx + tailwind-merge).
- Prefer standard Tailwind utilities over arbitrary values (`gap-2`, not `gap-[7px]`).
- Use design-token utilities from `app/styles.css` (`bg-primary`, `text-ink-muted`,
  `border-hairline-strong`, …); never raw hex.

## Folder structure

One folder per component, kebab-case name; the file matches the folder. No
`shared/ui/index.ts` barrel — import each component from its own folder.

```
shared/ui/
  <component-name>/
    <component-name>.tsx          // variants + props type + component (single file)
    <component-name>.stories.tsx
    index.ts                      // export * from './<component-name>'
```

- Folder/file: **kebab-case** (`filter-dropdown/filter-dropdown.tsx`).
- Exported component: **PascalCase** (`FilterDropdown`).
- Keep `cva` variants, the props `type`, and the component in the single `.tsx` file.
- Every component folder carries its own `AGENTS.md` (plus a `CLAUDE.md` holding only
  `@AGENTS.md`) with four parts: what the component is for and its Storybook path, **Usage**,
  **Props**, **Semantic styles** (variants and the design tokens behind them). Keep it in
  sync when props or variants change.

## Component rules

- Component is a **`React.FC`** with a `type` props alias.
- **Do not export** the `cva` variants object or the props type — keep them module-private.
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

Every component ships a story. Keep **one story per component** whenever possible — if
you think a component needs more than one, **ask first**.

Infer the story shape from `button/button.stories.tsx`:

- `StoryArgs` reuses the component's props via `Pick<ComponentProps<typeof X>, …>` for
  real props, plus story-only fields (e.g. `label`, `icon`) when needed.
- `meta.title` = `'<Group>/<Component>'` (see [Groups](#groups)); drive everything through
  `argTypes` + `args` and a single `render`.
- Export one story `Default` with `name: '<Component>'` (so a single-story component
  shows no extra nesting under its title).

### Groups

Stories are grouped by what the component is for; there is no common root. Pick the group
that fits, and add a new one only when none does — then also add it to `storySort.order`
in `.storybook/preview.tsx`, which fixes the sidebar order of the groups.

| Group | Holds |
|---|---|
| `Actions` | Things the user clicks to do something: buttons |
| `Inputs` | Form controls that take a value |
| `Content` | Presentation of data: badges, code, tables, cards (`Content/Card/<Kind>`), sink and operation badges |
| `Charts` | Charts |
| `Placeholders` | What stands in for content that is absent, failed or still loading: placeholders, errors, progress |
| `Overlays` | Layers above the page: sheets, tooltips, menus, toasts |
| `Layout` | Page structure and navigation within it: header, sidebar, tabs, toolbar |
| `Flows` | The flow diagrams |

`Colors` (`colors.mdx`) sits at the top level, outside any group.

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
