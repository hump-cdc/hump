# pages/configuration — how this page hangs together

`/configuration`, mounted by `app/routes/_sidebar/configuration.tsx`: the configuration hump was
started with, shown read-only. The route owns nothing but `component` and `staticData.title`
(the header title, read by `widgets/sidebar`); there are no search params and no loader.

## Layout

| File | Role |
|---|---|
| `index.ts` | public contract: `Configuration` |
| `ui/configuration.tsx` | a sized, padded box around `SourceViewer` from `@/entities/config/source-viewer` |

## The page owns only the height

Everything visible — the file path, the copy button, the highlighted source — is `SourceViewer`,
which reads `rawConfig` and `configPath` from the config query (`entities/config/AGENTS.md`). The
page holds no state.

What the page does decide is the box: `h-[calc(100svh-3.5rem)]`, where the other pages use
`flex-1`. `3.5rem` is the app header (`h-14` in `shared/ui/header/header.tsx`), so the box is
exactly the viewport below the header. `SourceViewer` is `min-h-0 flex-1 overflow-hidden` and its
`Code` block is `overflow-auto`, so with a fixed-height parent a long config scrolls inside the
card and the page itself does not grow. Change the header height and this constant goes with it.
