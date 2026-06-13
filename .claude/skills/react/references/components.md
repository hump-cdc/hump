# React Components, Hooks & Styling

## Component Structure

```typescript
// Order: props type → component → exports
type TaskListProps = {
    tasks: Task[];
    onSelect: (id: string) => void;
};

const TaskList: FC<TaskListProps> = ({ tasks, onSelect }) => {
    // implementation
};

export { TaskList };
```

**Rules:**
- Arrow functions with `FC` type annotation
- Named exports only (no default exports)
- Props type named `{ComponentName}Props` — never generic `Props`
- Define props as `type` (not interface); destructure in parameters
- Order within file: props type → component → sub-components → co-located hooks → exports

## Generic Components

Use explicit generic syntax with `ReturnType<FC>` return type (cannot use `FC<Props>` with generics):

```typescript
type ListProps<T extends string> = {
    options: ListOption<T>[]
    value: T
    onChange: (value: T) => void
}

const List = <T extends string>({ options, value, onChange }: ListProps<T>): ReturnType<FC> => {
    // ...
}

export { List, type ListProps }
```

**Rules:**
- Generic constraint on the type parameter (`T extends string`)
- Return type `ReturnType<FC>` instead of `FC<Props>` annotation
- Props type is also generic: `ListProps<T>`

## Ref as Prop (React 19)

React 19 deprecated `forwardRef`. Pass `ref` as a regular prop:

```typescript
type InputProps = ComponentProps<'input'> & {
    value: string
    onChange: (value: string) => void
}

const Input: FC<InputProps> = ({ ref, ...props }) => {
    return <input ref={ref} {...props} />
}

export { Input }
```

**Rules:**
- No `forwardRef` wrapper — `ref` is a standard prop in React 19
- No `displayName` needed (named arrow functions are self-documenting)
- Type `ref` via `ComponentProps<'element'>` which already includes it

## Extending Native HTML Props

Use `ComponentProps<'element'>` to inherit all native HTML attributes:

```tsx
import { Slot } from '@radix-ui/react-slot'

type ButtonProps = ComponentProps<'button'> & VariantProps<typeof buttonVariants> & {
    asChild?: boolean
}

export const Button: FC<ButtonProps> = ({ className, variant, size, asChild = false, ...props }) => {
    const Comp = asChild ? Slot : 'button'
    return <Comp data-slot="button" className={cn(buttonVariants({ variant, size, className }))} {...props} />
}
```

**Rules:**
- Extend `ComponentProps<'element'>` for the rendered HTML element
- Consumers get full native prop support (aria, data, event handlers) without manual forwarding
- Intersect with `VariantProps` and custom props

## Polymorphic Components (`asChild` + Slot)

Use `asChild` prop with a Slot primitive for render delegation — the consumer controls the rendered element:

```tsx
import { Slot } from '@radix-ui/react-slot'

type CardProps = ComponentProps<'div'> & { asChild?: boolean }

export const Card: FC<CardProps> = ({ asChild = false, className, ...props }) => {
    const Comp = asChild ? Slot : 'div'
    return <Comp data-slot="card" className={cn('rounded-lg border bg-card', className)} {...props} />
}

// Usage — renders as <div>
<Card>Content</Card>

// Usage — renders as <section>, merging Card's props onto it
<Card asChild><section>Content</section></Card>
```

**When to use `asChild`**: Trigger elements (dialog triggers, tooltip triggers), components that need to render as links/buttons interchangeably.

### `data-slot` Convention

Add `data-slot="component-name"` to root elements for CSS targeting and debugging:

```tsx
<div data-slot="card" className={cn(cardVariants({ variant }), className)} {...props}>
    <div data-slot="card-header" className={...}>...</div>
    <div data-slot="card-content" className={...}>...</div>
</div>
```

## Custom Hooks

- Extract when logic is used 3+ times, or when stateful logic is complex (even if used once)
- Name with `use` prefix: `useTaskFilter`, `useFileWatcher`
- Return objects for multiple values (not arrays, unless order is semantically meaningful like `useState`)
- Co-locate within the same FSD feature/slice

```typescript
const useTaskFilter = (tasks: Task[]) => {
    const [filter, setFilter] = useState('');
    const filtered = useMemo(
        () => tasks.filter(t => t.subject.includes(filter)),
        [tasks, filter]
    );
    return { filtered, filter, setFilter };
};
```

**Hook rules:**
- Follow Rules of Hooks (no conditionals, no loops at top level)
- Explicit dependency arrays (no suppressed exhaustive-deps)
- `useCallback` only when passing callbacks to memoized children — profile first
- `useMemo` only for expensive computations — profile first
- `useRef` for: values that don't trigger re-renders, DOM references, previous values

## Hook Layering Pattern

Build hooks in layers from simple to complex:

```typescript
// Layer 1: Base data hook (raw data)
useTasksData()           // → TasksData { list, map }

// Layer 2: Derived (applies UI state)
useTasks()               // → filtered & sorted Task[]

// Layer 3: Composite (combines data + selection state)
useSelectedTask()        // → single Task | undefined

// Layer 4: Aggregation (computes statistics)
useProgress()            // → { pending, inProgress, completed, blocked }
```

## Styling with CVA + Tailwind v4

### CVA (Class Variance Authority) — Primary Variant Approach

Use CVA for any component with visual variants.

Quick rules:
- Define `*Variants` const adjacent to component (same file)
- Use `VariantProps<typeof variantsFn>` for prop types — keeps variants and types in sync
- Pass `className` as last arg to CVA call for consumer overrides
- Always wrap CVA output with `cn()` for Tailwind conflict resolution
- Always set `defaultVariants`
- Use `compoundVariants` for combination-specific styles instead of conditional JSX logic

### `cn()` Utility — Class Merging

Always use `cn()` (not raw `clsx`) for class composition. It wraps `clsx` + `twMerge` to resolve Tailwind conflicts.

Canonical import: `import { cn } from '@/shared/utils'`

```tsx
// shared/utils/cn.ts
import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs))
```

```tsx
// cn() resolves conflicts — last value wins
cn('px-4 py-2', 'px-2')  // → 'py-2 px-2' (px-4 resolved to px-2)

// Conditional classes
<div className={cn('flex items-center gap-2', isActive && 'ring-2 ring-blue-500', className)} />
```

**When to use which:**
- **CVA + `cn()`**: Component has named variants (size, intent, state) → always CVA, pass `className` through `cn()`
- **`cn()` alone**: One-off conditionals without reusable variants
- **Both**: CVA for variants + `cn()` for additional one-off logic

**Rule**: Never use `clsx` directly — always use `cn()` to ensure Tailwind conflict resolution.

**Patterns:**
- Accept `className` prop for consumer override (pass to CVA or `cn()`)
- Use `data-*` attributes with Radix UI for state-based styling

## Dynamic Component Resolution

Use record mapping for state-based rendering:

```tsx
const states: Record<Dto.Access, ComponentType> = {
    Unknown: RequestAccessRequest,
    Requested: RequestAccessRequested,
    Approved: RequestAccessApproved,
    Rejected: RequestAccessRejected,
}

const Content = !access ? RequestAccessRequest : states[access]
return <Content />
```

## React 19 Hooks

### `use()` — Read Promises and Context in Render

Reads a Promise or Context value. Unlike other hooks, can be called conditionally:

```tsx
import { use } from 'react'

const Comments: FC<{ commentsPromise: Promise<Comment[]> }> = ({ commentsPromise }) => {
    const comments = use(commentsPromise)  // suspends until resolved
    return comments.map(c => <p key={c.id}>{c.text}</p>)
}

// Parent wraps with Suspense
<Suspense fallback={<Skeleton className="h-20 w-full" />}>
    <Comments commentsPromise={fetchComments()} />
</Suspense>
```

**Rules**: Promise must be stable (passed as prop or from cache) — not created in render.

### `useActionState()` — Async Form Actions

Manages state + pending status for form actions (replaces deprecated `useFormState` from `react-dom`):

```tsx
import { useActionState } from 'react'

const [state, formAction, isPending] = useActionState(
    async (previousState, formData) => {
        const result = await submitForm(formData)
        return result
    },
    initialState,
)
```

### `useOptimistic()` — Optimistic UI Updates

Renders provisional UI during async mutations, auto-reverts on failure:

```tsx
import { useOptimistic } from 'react'

const [optimisticItems, addOptimistic] = useOptimistic(
    items,
    (state, newItem) => [...state, { ...newItem, pending: true }],
)
```

**When to use**: Use `useOptimistic` to show provisional UI during async mutations that auto-reverts on failure.

## Export Patterns

Every slice exports through `index.ts`:

```tsx
// entities/applications/applications-list/index.ts
export { ApplicationsList } from './ui/applications-list'

// features/licenses/issue-license/index.ts
export { IssueLicense } from './ui/issue-license'
```

**Rules:**
- Only export what the consuming layer needs
- Separate type exports from value exports
- Never import from internal paths of another slice (use barrel only)

## Storybook

See `src/shared/ui/CLAUDE.md` for story file structure, controls/args conventions, event-handler actions, and dark mode verification.

## Prohibited Patterns

- Class components
- `forwardRef` (deprecated in React 19 — use `ref` as a regular prop)
- `React` namespace imports (`React.FC`, `React.useState`)
- Default exports
- Prop drilling > 2 levels (use React Context)
- Side effects in render (use `useEffect`)
- Mutating props or state directly
- Index as key in lists (use stable IDs)
- Mixing concerns in one component (extract hooks/sub-components)
- CSS modules or styled-components (use Tailwind + CVA)
- Raw `clsx` without `twMerge` (use `cn()`)
