---
name: react
description: >
  React development patterns with Feature-Sliced Design, built on TanStack Router and rsbuild.
  ALWAYS use when working with React: (1) Creating or modifying pages and route files
  (TanStack Router file-based routing), (2) Building and composing React components,
  (3) Custom hooks and local/context state, (4) Design system components and styling
  (CVA + Tailwind v4), (5) Feature-Sliced Design (FSD) folder structure and conventions,
  (6) Adding or customizing shadcn/ui components (button, dialog, table, dropdown, etc.),
  (7) Storybook stories for the UI kit.
  Auto-triggers for all React operations. Also triggers when user mentions shadcn,
  Radix UI, TanStack Router, rsbuild, or asks to create/add UI components like button, dialog, input, select, table, etc.
---

# React Developer

Build scalable React applications with TanStack Router (file-based routing) and rsbuild, following Feature-Sliced Design architecture.

## Stack

- **React 19** + TypeScript
- **TanStack Router** — file-based routing in `app/routes/`
- **rsbuild** — bundler (`rsbuild dev` / `rsbuild build`)
- **Tailwind CSS v4** + **CVA** + **shadcn/ui** (Radix UI primitives) for styling
- **lucide-react** — icons
- **Storybook** — UI kit documentation
- **Biome** — format/lint; **Bun** — package manager
- Path alias: `@/*` → `src/*`

## Core Principles

- **Components**: Arrow functions with `FC` type, named exports, `{ComponentName}Props` type
- **Hooks**: Extract at 3+ uses or complex stateful logic; return objects not arrays
- **State**: Local (`useState`); cross-cutting via React Context (e.g. theme through `@/shared/theme`)
- **FSD Layers**: `app → pages → widgets → features → entities → shared` (strict downward imports)
- **Performance**: Profile before optimizing; composition over memo
- **Routing**: TanStack Router file-based routes in `app/routes/`; regenerate the route tree with `tsr generate` (or the rsbuild router plugin in dev)
- **Tailwind CSS**: CSS-first config via `@theme`; style with utility classes and design-token utilities; no CSS modules or styled-components
- **CVA**: Primary approach for variant-based styling; always use `cn()` (`clsx` + `twMerge`) for class merging — never raw `clsx`
- **shadcn/ui**: Use shadcn components (Radix UI primitives + Tailwind styling) for accessible design system
- **Dark mode**: Support via the custom `@/shared/theme` provider with `dark:` Tailwind prefix
- **Icons**: `lucide-react` for all SVG icons — no custom SVGs or other icon libraries
- **Named exports only**: No default exports — explicit imports improve discoverability

## Reference Files

Read only what's needed for the current task:

| File | Read when… |
|------|-----------|
| [components.md](references/components.md) | Writing components and hooks; component structure and typing; Tailwind v4; `cn()` utility; `asChild`/Slot pattern; hook layering; React 19 hooks |
| [react-performance.md](references/react-performance.md) | Optimizing renders; deciding when to use `React.memo`; error handling patterns; loading/error UI components |

For `shared/ui/` design-system conventions (component folder structure, CVA variants, Storybook stories), see `src/shared/ui/CLAUDE.md`.

## Common Workflows

### Layer Selection

Pick the correct FSD layer before creating anything:

| You're building… | Layer | Can import from |
|-------------------|-------|-----------------|
| Reusable UI primitive (button, input, dialog) | `shared/ui/` | nothing (no FSD imports) |
| Domain model or entity-scoped UI | `entities/` | shared |
| User interaction (create, edit, delete, filter) | `features/` | entities, shared |
| Composite page section combining features/entities | `widgets/` | features, entities, shared |
| Full screen bound to a route | `pages/` | widgets, features, entities, shared |

Every slice exposes a public API through `index.ts` — only export what consuming layers need; never import from another slice's internal paths.

### Creating a Design System Component (`shared/ui/`)

1. Create folder: `src/shared/ui/<component-name>/`
2. Add component file: `<component-name>.tsx` — install via shadcn CLI then adapt to project standards (see `src/shared/ui/CLAUDE.md`)
3. Add story: `<component-name>.stories.tsx` — cover all variants + dark mode
4. Export via component barrel: `<component-name>/index.ts` (`export * from './<component-name>'`) — no top-level `shared/ui/index.ts`
5. Import as `import { Button } from '@/shared/ui/button'`
6. Verify: light/dark theme, keyboard navigation, `data-slot` attribute on root

### Creating a Page (`pages/`)

1. Create FSD page slice: `src/pages/<page-name>/`
2. Add `ui/` segment — compose widgets/features, wrap async components in `<ErrorBoundary>` + `<Suspense>`
3. Export public API via `index.ts`
4. Create TanStack Router route file under `app/routes/` — a thin wrapper importing the FSD page component, then regenerate the route tree (`tsr generate`)
5. Imports allowed: `widgets/`, `features/`, `entities/`, `shared/`

```tsx
// app/routes/applications/index.tsx — thin TanStack Router file route
import { createFileRoute } from '@tanstack/react-router'

import { ApplicationsList } from '@/pages/applications/applications-list'

const Page = () => <ApplicationsList />

export const Route = createFileRoute('/applications/')({ component: Page })
```

```tsx
// With params — app/routes/licenses/$id.tsx
import { createFileRoute } from '@tanstack/react-router'

import { LicenseDetails } from '@/pages/licenses/license-details'

const Page = () => {
    const { id } = Route.useParams()
    return <LicenseDetails id={id} />
}

export const Route = createFileRoute('/licenses/$id')({ component: Page })
```
