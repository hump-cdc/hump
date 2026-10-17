# pages/flows — how this page hangs together

`/`, the index route — mounted by `app/routes/_sidebar/index.tsx`, which sets only `component` and
`staticData.title`. No search params, no loader. The page is the topology canvas with two things
floating over it, and its one job is to tell the canvas how much of itself they cover.

## Layout

| File | Role |
|---|---|
| `index.ts` | public contract: `Flows` |
| `ui/flows.tsx` | composes canvas, toolbar and inspector; positions the toolbar |
| `ui/use-insets.ts` | measures the overlays and turns them into `ViewportInsets` for the canvas |

## Composition

All three parts come from `entities/flows` (see `entities/flows/AGENTS.md`):

- `Flows` from `@/entities/flows/flows`, imported as `FlowsCanvas` because the page has the same
  name. It takes `overlay` and renders it **inside** its own positioned, clipped region — that is
  why the toolbar wrapper and the inspector can be `absolute` and still dock to the canvas.
- `FlowsToolbar` from `@/entities/flows/toolbar` — the table and sink pickers.
- `FlowInspector` from `@/entities/flows/inspector` — the details sheet for the selected node or
  edge; `useFlowInspectorStore` from the same slice is the selection.

The page holds no state of its own. Selection lives in the inspector store (in memory); the
toolbar filter and the summary/detailed mode live in stores the canvas slice persists.

## Insets

The canvas pans to a selected item and must not park it under an overlay, so `useInsets` reports
the covered margins as `viewportInsets` (`ViewportInsets` from `@/shared/ui/flow`):

| Inset | Value | Source |
|---|---|---|
| `top` | toolbar wrapper height | measured (`useElementSize`); the wrapper's `p-6` is included |
| `right` | 300 with the desktop inspector open, else 0 | the constant `INSPECTOR_WIDTH` |
| `bottom` | inspector height with the mobile inspector open, else 0 | measured via `inspectorRef` |
| `left` | 0 | — |

The right inset is a constant on purpose: the width is known up front, so the inset changes in
the same render as the selection it is revealed for. The mobile sheet's height depends on its
content, so that one is measured. "Mobile" is `useIsMobile()` from `@/shared/utils`.

## Three numbers that move together

The desktop inspector width is written three times, with nothing tying them together:

- `w-75` on the sheet in `entities/flows/inspector/ui/flow-inspector.tsx`;
- `INSPECTOR_WIDTH = 300` in `use-insets.ts`;
- `pr-81` on the toolbar wrapper in `flows.tsx` — 324px, the 300px inspector plus the wrapper's
  own `p-6`.

The toolbar's padding transition is 300ms when it shrinks and 200ms when it grows back, matching
the open and close animations of the shared sheet (`shared/ui/sheet/sheet-content.tsx`).

## Gotchas

- The toolbar wrapper spans the full canvas width and is `pointer-events-none`; only its children
  re-enable pointer events (`*:pointer-events-auto`). Anything added beside `FlowsToolbar` inside
  the wrapper captures clicks; the wrapper's padding does not.
- `toolbarRef` sits on the padded wrapper, not on the toolbar, so the top inset already includes
  the gap below the toolbar.
- `use-insets.ts` is in `ui/`, not `model/`: it is layout measurement, with no data in it.
