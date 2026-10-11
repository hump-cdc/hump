# shared/theme — how this module hangs together

Light/dark theming: one class on `<html>`, a persisted choice, and a React context around both.
The colours themselves are not here — they are CSS custom properties in `app/styles.css`. FSD
layer: `shared`; the module imports nothing from the app, only `react`.

Consumers: `src/app/providers.tsx` mounts `ThemeProvider`; `widgets/sidebar`
(`app-sidebar-theme.tsx`) calls `useTheme`; `.storybook/preview.tsx` wraps stories in
`ThemeProvider`; `rsbuild.config.ts` imports `theme-script.ts` by relative path.

## Layout

| File | Role |
|---|---|
| `types.ts` | `Theme` = `'light' \| 'dark' \| 'system'`, `ResolvedTheme` = `'light' \| 'dark'` |
| `constants.ts` | storage key, default theme, class names, transition length, media query |
| `theme-script.ts` | `themeInitScript`: the pre-paint script, as a string |
| `theme-provider.tsx` | `ThemeProvider`, `ThemeContext`, reading/persisting/applying the theme |
| `use-theme.ts` | `useTheme`: the context, or a throw outside a provider |
| `index.ts` | `ThemeProvider`, `useTheme`, and the types `Theme`, `ResolvedTheme`, `ThemeContextValue` |

The constants, `ThemeContext` and `themeInitScript` are not part of the public contract.

## How a theme reaches the screen

`app/styles.css` defines semantic tokens (`--canvas`, `--ink`, …) on `:root` for light and
re-points them under `.dark`; `@theme inline` exposes them as utilities (`bg-canvas`, `text-ink`).
`@custom-variant dark (&:where(.dark, .dark *))` ties Tailwind's `dark:` to the same class, and
`color-scheme` flips with it. So the whole job of this module is to put `dark` on
`document.documentElement`, or take it off. There is no `light` class and no data attribute.

Two writers set that class, at different times:

1. **Before first paint — `themeInitScript`.** `rsbuild.config.ts` injects it through `html.tags`
   as an inline `<script>` in `<head>`, placed before the other tags (`head: true, append:
   false`), so it runs synchronously while the app bundle is still loading. It reads
   `localStorage`, falls back to the default, resolves `system` through `matchMedia`, and toggles
   the class. Everything is inside `try/catch`.
2. **On change — `ThemeProvider`.** `setTheme` stores the choice, resolves it, and calls
   `applyResolvedTheme`. A `matchMedia` `change` listener does the same while the theme is
   `system`, and is attached only then.

**The provider never touches the DOM on mount.** It initialises its state from `localStorage` and
trusts the script to have applied the same value. Remove the script and a stored `dark` renders
light until the user switches.

## The script duplicates the constants — keep them in sync

`theme-script.ts` is a string evaluated in the page, so it cannot import anything. It repeats
four literals from `constants.ts`: `'theme'` (`THEME_STORAGE_KEY`), `'dark'` as the fallback
(`DEFAULT_THEME`), `'dark'` as the class (`DARK_CLASS`), and `'(prefers-color-scheme: dark)'`
(`DARK_MEDIA_QUERY`). The validation of the stored value (`light`/`dark`/`system`, anything else
→ default) is duplicated too, in `readStoredTheme`. Changing any of these means editing both
files; nothing checks it.

The file is also imported by `rsbuild.config.ts`, i.e. it is loaded by the build tool, outside the
app bundle. Keep it free of imports.

## Behaviour worth knowing

- Default is `dark` — with nothing stored, the OS preference is **not** consulted. `system` is
  honoured when stored, but the sidebar toggle cycles only `light` ↔ `dark`
  (`THEME_CYCLE` in `widgets/sidebar/app-sidebar-theme.tsx`), so nothing in the UI sets it today.
- `theme` is the choice, `resolvedTheme` what is actually shown. Read `resolvedTheme` when code
  needs to know light or dark; `theme` can be `system`.
- `localStorage` failures are swallowed on read and on write: the choice then lasts for the
  session only.
- Switching adds `theme-transition` to `<html>` for `THEME_TRANSITION_MS` (250 ms). The class
  forces a 0.2 s transition of `background-color`, `border-color`, `color`, `fill` and `stroke`
  on every element with `!important` (`app/styles.css`). The initial paint gets no such class, so
  it does not animate. The CSS duration and the constant are two separate numbers, not derived
  from each other.
- `prefersDark` guards `window`; nothing else in the provider does.

## Storybook

`.storybook/preview.tsx` does not go through this module's class handling. Its `withTheme`
decorator renders one or two panels (the default is side by side) and puts `dark` on the panel
`<div>` itself; because the tokens and the `dark:` variant are keyed on `.dark` and its
descendants, a subtree themes independently of `<html>`. Each panel is still wrapped in
`ThemeProvider`, so components that call `useTheme` render — but the context value there comes
from `localStorage`, not from the panel, and says nothing about which panel a component is in.
A component that must look right in Storybook's dark panel has to get its colours from tokens
or `dark:` utilities, not from `resolvedTheme`.
