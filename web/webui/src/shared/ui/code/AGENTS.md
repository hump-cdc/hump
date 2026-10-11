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

The folder exports `Code` only. `code.grammar.ts` (`detectGrammar`, `formatCode`, the
`CodeGrammar` type) is internal and not re-exported from `index.ts`.

## Props

No HTML attributes pass through; only the props below.

| Prop | Type | Default | Description |
|---|---|---|---|
| `value` | `string` | required | Source text |
| `lang` | `'yaml' \| 'json' \| 'sql' \| 'plaintext'` | — | Grammar to use; skips detection **and** formatting |
| `lineNumbers` | `boolean` | `true` | Line-number gutter |
| `format` | `boolean` | `false` | Pretty-print JSON (2-space indent) and YAML (block style, 2-space indent, comments kept) before highlighting |
| `className` | `string` | — | Wrapper class; sizing is the caller's job |

Grammar detection, when `lang` is not set: JSON if the text parses as JSON (checked first, since
JSON is also YAML); YAML if it parses without errors into a map or a list; otherwise plain text,
bare scalars included. SQL is never detected — pass `lang="sql"`. `format` is ignored when `lang`
is set.

## Semantic styles

No cva variants. The wrapper is `hump-code min-h-0 flex-1 overflow-auto`: it fills a flex parent
and scrolls inside it, so give it a bounded box. The `.hump-code` rules in `app/styles.css` map
TanStack Highlight's `th-*` token classes onto the `--syntax-*` tokens, which flip between light
and dark:

| Token | Paints |
|---|---|
| `--syntax-bg` | background (`--surface-1` by default) |
| `--syntax-fg` | default text |
| `--syntax-tag` | properties, functions |
| `--syntax-string` | strings |
| `--syntax-number` | numbers |
| `--syntax-keyword` | keywords, literals |
| `--syntax-comment` | comments (italic) |
| `--syntax-punctuation` | operators |
| `--hump-code-ln` | line numbers (`--ink-tertiary`) |

Text is JetBrains Mono, 13px. To change the background from a call site, override the token:
`className="[--syntax-bg:var(--canvas)]"`.
