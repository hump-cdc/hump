# pages/operations — how this page hangs together

`/operations`: the delivery log with its filter and a details drawer. The filter, list and drawer
are the three slices of `entities/operations` (`entities/operations/AGENTS.md`); the page passes
them no props, and they meet in the stores. The page reads only "is something selected" for the
layout and writes the filter only through `useSearchFilter`.

It is the one page with a `model/`, because the URL can preset the filter. `index.ts` also exports
`operationsSearchSchema`: the route file sets it as `validateSearch`, while the page defines what
is valid.

## The search params are a one-shot preset

The URL is a way **in**, not the filter's home. `useSearchFilter` runs an effect on the validated
search:

1. `toFilterValues(search)` builds a whole filter; if it is defined, `apply` writes it.
2. If any schema key is present, it navigates to `search: {}` with `replace: true`.

So after a link lands, the URL is clean and the filter is in the persisted store: a reload or Back
keeps the filter as the user has since changed it instead of re-applying the link. The replace
passes `viewTransition: false`, since the router's `defaultViewTransition` would otherwise animate
a URL cleanup.

The preset is **exclusive**. `toFilterValues` spreads `defaultFilterValues` first, so every facet
the params leave out goes back to "All" and the search text is cleared: the page lists exactly what
the link names. With no usable param it returns `undefined` and the stored filter is left alone.

## Params

| Param | Filter facet | Checked against |
|---|---|---|
| `table` | `tables` | — |
| `sink` | `sinks` | — |
| `target` | `targets` | — |
| `operation` | `operations` | `operationItems` |
| `result` | `statuses` | `statusItems` |

- Each accepts a single value or a repeated key and always comes out as `string[]`.
- The router's default search parser JSON-decodes values, so `target=123` arrives as a number;
  `facetSchema` reads it back as a string.
- Values are never comma-split: a target can be a URL.
- A malformed facet is dropped (`.catch(undefined)`) rather than failing the route.
- `operation` and `result` keep only values the filter entity knows, and a facet left with none
  counts as absent. A new option in `@/entities/operations/filter` is accepted here without a
  change. `table`, `sink` and `target` pass through unchecked: their options only arrive with the
  config and metrics.

## Changing a param touches other slices

- The only in-app producer of these params is `entities/flows/inspector/model/operations-search.ts`
  ("View in Operations"), which declares its own `OperationsSearch` type with `table`, `sink` and
  `target`. Rename a key here and change it there. Other links to `/operations` carry no search.
- A new facet needs three edits here: the key in `operationsSearchSchema`, the mapping in
  `toFilterValues`, and the `if` that decides whether anything was preset. The URL cleanup needs
  none: it iterates `operationsSearchSchema.keyof()`.
- `useSearch` is bound to the route **id** `/_sidebar/operations`, `useNavigate` to the **path**
  `/operations`. Moving the route out of the `_sidebar` layout changes the first, not the second.

## Making room for the drawer

`OperationDetails` is a non-modal sheet that, on desktop, is `absolute` at `w-95` (380px). The grid
region is `relative overflow-hidden` and holds both the list and the drawer. While an operation is
selected on desktop the region gets `pr-99` (396px: the drawer plus a 16px gap), so the list
shrinks beside the drawer instead of sitting under it. `pr-99` and the entity's `w-95` must change
together.

The padding transition is 300ms in and 200ms out, the open and close animations of
`shared/ui/sheet/sheet-content.tsx`. On mobile (`useIsMobile()`) there is no padding: the drawer
portals to the viewport as a bottom sheet and overlays the list.
