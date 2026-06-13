# React: Performance & Error Handling

## Performance

**Profile before optimizing.** Measure actual bottlenecks first.

### Composition Over Memo

Split components to isolate frequently updating parts, pass stable children:

```typescript
// BAD: creates new object on every render
<TaskList config={{ showCompleted: true }} />

// GOOD: stable reference
const CONFIG = { showCompleted: true };
<TaskList config={CONFIG} />

// GOOD: composition isolates re-renders
const ExpensiveList = ({ children }) => {
    // expensive logic
    return <>{children}</>;
};
<ExpensiveList>
    <FrequentlyUpdatingComponent />
</ExpensiveList>
```

### When to Use `React.memo`

Only after profiling shows unnecessary re-renders:
- Expensive components that receive stable props
- List items in large lists
- Not needed for most components — composition is usually better

### Optimization Rules

```typescript
// useMemo for derived data and transformations
const options = useMemo(
    () => items.map((task) => ({ id: task.id, title: task.subject })),
    [items],
)

// useCallback for callbacks passed as props to memoized children
const handleSelect = useCallback(
    (id: string) => setSelectedId(id),
    [setSelectedId],
)

// useRef for values that shouldn't trigger re-renders
const callbackRef = useRef(onChanged)
useEffect(() => { callbackRef.current = onChanged }, [onChanged])
```

## Error Handling

### Three-Layer Strategy

1. **ErrorBoundary** — catches unhandled component tree errors
2. **Suspense** — handles loading states
3. **Component-level** — handles known error states from data loading

```typescript
// Page level: boundary + suspense
<ErrorBoundary>
    <Suspense fallback={<LoadingPlaceholder />}>
        <DataComponent />
    </Suspense>
</ErrorBoundary>
```

### Component-Level Guards

- Early returns for missing data: `if (!data) return null`
- Optional chaining for nullable refs: `ref?.current?.method()`
- Nullish coalescing for defaults: `const tasks = data?.list ?? []`
- No try-catch in components — rely on error boundaries and type safety

### Loading & Error UI Components

**Skeleton placeholders** — used as `Suspense` fallbacks and loading states:

```tsx
// Skeleton mimics the shape of the content being loaded
<Skeleton className="h-4 w-[200px]" />
<Skeleton className="h-10 w-full rounded-md" />

// Compose skeletons to match actual layout
const UserCardSkeleton: FC = () => (
    <div className="flex items-center gap-3 p-4">
        <Skeleton className="size-10 rounded-full" />
        <div className="flex flex-col gap-2">
            <Skeleton className="h-4 w-[150px]" />
            <Skeleton className="h-3 w-[100px]" />
        </div>
    </div>
)
```

**Inline loading spinner** — used inside buttons and small containers:

```tsx
import { LoaderCircle } from 'lucide-react'

<Button disabled={isSubmitting}>
    {isSubmitting && <LoaderCircle className="animate-spin" />}
    Save
</Button>
```

**Error card** — used for entity-level error display:

```tsx
const CardError: FC<{ title: string; message?: string }> = ({ title, message }) => (
    <Card className="border-destructive">
        <CardHeader>
            <CardTitle className="text-destructive">{title}</CardTitle>
        </CardHeader>
        {message && <CardContent><p className="text-sm text-muted-foreground">{message}</p></CardContent>}
    </Card>
)
```

**Conventions**:
- Skeleton shape should match the content it replaces
- Always show spinner inside the triggering button (not a separate overlay)
- Use `CardError` for recoverable entity errors; `ErrorBoundary` for unrecoverable crashes
- Icons from `lucide-react` (`LoaderCircle`, `AlertCircle`, etc.)

## Code Quality

- Prefer early returns over nested conditionals
- Extract magic numbers/strings to named constants
- Descriptive variable names (no single letters except loop indices)
