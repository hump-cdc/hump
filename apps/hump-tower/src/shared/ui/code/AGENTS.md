# Code

A read-only, syntax-highlighted viewer for YAML, JSON, SQL and plain text, built on TanStack
Highlight. Use it for config sources, payloads and queries; for a short inline token in running
text pick `@/shared/ui/mark`.
Storybook: `Content/Code`.

## Usage

```tsx
import { Code } from '@/shared/ui/code'

<Code className="h-full w-full" value={yamlSource} />
<Code value={payload} format lineNumbers={false} className="[--syntax-bg:var(--canvas)]" />
<Code value={query} lang="sql" lineNumbers={false} />
```

The folder exports `Code` only; `code.grammar.ts` (`detectGrammar`, `formatCode`) is internal.

## Props

No HTML attributes pass through.

| Prop | Description |
|---|---|
| `lang` | Grammar to use; skips detection **and** `format` |
| `format` | Pretty-print JSON (2-space indent) and YAML (block style, 2-space indent, comments kept) before highlighting |

Grammar detection, when `lang` is not set: JSON if the text parses as JSON, scalars such as `42`,
`true` or `"x"` included (checked first, since JSON is also YAML); YAML if it parses without errors
into a map or a list, so a bare YAML scalar (`hello`) is not YAML; otherwise plain text. SQL is
never detected — pass `lang="sql"`.

## Semantic styles

No cva variants. The wrapper is `hump-code min-h-0 flex-1 overflow-auto`: it fills a flex parent
and scrolls inside it, so give it a bounded box. The `.hump-code` rules in `app/styles.css` map
TanStack Highlight's `th-*` classes onto the `--syntax-*` tokens, which flip between light and
dark; `--syntax-bg` defaults to `--surface-1`. To change the background from a call site,
override the token: `className="[--syntax-bg:var(--canvas)]"`.
