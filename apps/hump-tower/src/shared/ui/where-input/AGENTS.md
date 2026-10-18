# WhereInput

An input for a PostgreSQL `WHERE` condition: one line while idle, a multi-line editor with syntax
highlighting and autocompletion while focused. Use it only for SQL conditions
(`features/snapshot`, the per-table row filter); for plain text use `TextInput`
(`@/shared/ui/text-input`).
Storybook: `Inputs/Where Input`.

Exports: `WhereInput` and the type `WhereField`.

## Usage

```tsx
import { WhereInput } from '@/shared/ui/where-input'

<WhereInput
    value={draft}
    onValueChange={setDraft}
    onBlur={() => onWhereCommit(draft)}
    suggestion={table.suggestedWhere}
    fields={table.columns}
    disabled={!selected}
    invalid={Boolean(error)}
/>
```

## Props

Only the props below; nothing else is passed through.

| Prop | Type | Default | Description |
|---|---|---|---|
| `value` | `string` | required | The condition, without the `WHERE` keyword. Controlled only |
| `onValueChange` | `(value: string) => void` | required | Called on every edit, on clear (`''`) and on restore (`suggestion`) |
| `onBlur` | `() => void` | — | Called when the editor loses focus, e.g. to act on the finished condition |
| `placeholder` | `string` | `'(no filter)'` | |
| `suggestion` | `string` | — | Suggested condition offered for restore while the input is focused and empty |
| `fields` | `WhereField[]` | `[]` | Columns offered by autocompletion |
| `invalid` | `boolean` | — | Error styling; sets `aria-invalid` on the panel |
| `disabled` | `boolean` | — | Disables the textarea and makes the editor read-only |
| `className` | `string` | — | Merged into the outer wrapper (`relative h-9 w-full`), not the panel |
| `ref` | `Ref<HTMLTextAreaElement>` | — | Receives the editor's `<textarea>` |

`WhereField`:

| Field | Type | Description |
|---|---|---|
| `name` | `string` | Column name; quoted on insert unless it is a plain lower-case, non-reserved identifier |
| `type` | `string` (optional) | Column type as PostgreSQL formats it, shown next to the name |

## Behaviour

- **Expanding.** The outer wrapper keeps `h-9`; the panel is absolutely positioned over it. On
  focus it grows to the content, between 3 and 10 lines, and overlays what is below, so the parent
  layout never reflows. From `lg` up it also widens to at least `min-w-lg`. Below 1024px
  (`useIsMobile()`) it stays one line even while focused.
- **Collapsed.** Shows the first non-blank line, highlighted and ellipsized, plus `…` when more
  non-blank lines follow.
- **Autocompletion.** Opens while typing a word and on Ctrl+Space; offers `fields`, operators,
  functions and keywords depending on the position, never inside a string, comment or quoted
  identifier. ArrowUp/ArrowDown move, Enter or Tab accept, Escape closes the list (and blurs the
  editor when the list is closed).
- **Clear and restore.** Both buttons exist only while focused: `X` when there is a value, the
  restore arrow (tooltip shows `suggestion`) when the value is empty and `suggestion` is set.
- **Clicks.** The panel swallows `click`, so an enclosing `<label>` (a selectable table row) is not
  activated, and redirects `mousedown` on its chrome to the editor.
- **Value sync.** The editor (`prism-react-editor`) owns its text after mount; a `value` changed
  from outside is written back into it.
- **A11y.** The textarea is `role="combobox"` with `aria-expanded`, `aria-controls` and
  `aria-activedescendant` pointing at the `role="listbox"` popup. There is no label: the caller
  provides one.
- **Provider.** The restore button renders a `Tooltip` (`@/shared/ui/tooltip`), which needs
  `TooltipProvider` above — mounted in `@/app` `Providers`.

## Semantic styles

No cva variants. Part of the styling lives outside Tailwind, in the `.hump-where` rules of
`app/styles.css`; the heights in `where-input.consts.ts` (36px collapsed, 20px lines, 14px
vertical padding, 2px border) must stay in sync with them.

| Part | Tokens |
|---|---|
| panel | `border-hairline`, `bg-canvas`, `rounded-lg`; focused: `z-20`, from `lg` also `shadow-lg` |
| `WHERE` prefix | `text-ink-subtle`; `text-danger` when `invalid` |
| text / placeholder | `text-ink`, `font-mono` / `--ink-tertiary` |
| caret / selection | `--ink` / `--primary` at 25% |
| clear, restore buttons | `text-ink-subtle`; hover `text-ink-muted` (clear), `text-ink` (restore) |
| suggestion popup | `border-hairline`, `bg-canvas`, `rounded-xl`, `shadow-lg`; active option `bg-surface-1` |
| suggestion option | `text-ink`, `font-mono text-xs`; kind icon and type `text-ink-subtle`; matched characters `text-primary` |

Syntax colours (`.hump-where .token.*`):

| Token | CSS variable |
|---|---|
| operator, punctuation | `--syntax-punctuation` |
| keyword, boolean, word operators (`AND`, `IN`, …) | `--syntax-keyword` |
| string | `--syntax-string` |
| number | `--syntax-number` |
| comment | `--syntax-comment`, italic |
| function, cast, quoted identifier | `--syntax-tag` |

Suggestion kind icons: `field` → `Columns3`, `function` → `SquareFunction`, `keyword` →
`CaseSensitive`, `operator` → `Equal`.

States of the panel, the same as `TextInput`:

- focus-within: `ring-2 ring-primary` with `ring-offset-2 ring-offset-canvas`.
- invalid: `border-danger`, focus ring `ring-danger`.
- disabled: `opacity-50`, `pointer-events-none`.
