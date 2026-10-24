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

The props type in `where-input.tsx` is closed: nothing else is passed through. Not obvious from
the types:

- `value` is the condition without the `WHERE` keyword, and the input is controlled only.
- `onValueChange` fires on every edit, on clear (`''`) and on restore (with `suggestion`).
- `suggestion` is offered for restore while the input is focused and empty.
- `invalid` sets `aria-invalid` on the panel; `disabled` disables the textarea and makes the editor
  read-only.
- `className` goes to the outer wrapper (`relative h-9 w-full`), not the panel; `ref` receives the
  editor's `<textarea>`.
- `WhereField.name` is quoted on insert unless it is a plain lower-case, non-reserved identifier;
  `type` is shown next to it as PostgreSQL formats it.

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
- **Clicks.** The panel cancels `click`, so an enclosing `<label>` (a selectable table row) is not
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

The panel has the same look and states as `TextInput`: `border-hairline`, `bg-canvas`; focus ring
`ring-primary`; invalid `border-danger` (and the `WHERE` prefix turns `text-danger`); disabled
`opacity-50`. Focused it also gets `z-20`, and from `lg` `shadow-lg`. The caret is `--ink`, the
selection `--primary` at 25%.

Syntax colours (`.hump-where .token.*`):

| Token | CSS variable |
|---|---|
| operator, punctuation | `--syntax-punctuation` |
| keyword, boolean, word operators (`AND`, `IN`, …) | `--syntax-keyword` |
| string | `--syntax-string` |
| number | `--syntax-number` |
| comment | `--syntax-comment`, italic |
| function, cast, quoted identifier | `--syntax-tag` |

In the suggestion popup the matched characters are `text-primary`, the active option
`bg-surface-1`. Kind icons: `field` → `Columns3`, `function` → `SquareFunction`, `keyword` →
`CaseSensitive`, `operator` → `Equal`.
