# pages/snapshot — how this page hangs together

`/snapshot`, mounted by `app/routes/_sidebar/snapshot.tsx`, which sets only `component` and
`staticData.title`. No search params, no loader.

## Layout

| File | Role |
|---|---|
| `index.ts` | public contract: `SnapshotPage` |
| `ui/snapshot-page.tsx` | a width-capped, padded column around `Snapshot` from `@/features/snapshot` |

## A frame around the feature

This is the only page built on a **feature** rather than entities, and it adds nothing to it: no
props, no state, no hooks. Which screen shows (idle, processing, done), the action bars, the
stream of the run's state and the start/stop calls are all inside the feature — see
`features/snapshot/AGENTS.md`.

What the page contributes is the frame: `max-w-6xl`, `p-6`, and a `flex flex-1 flex-col` column.
The feature's root is `flex-1` and relies on that column to fill the height.

## Gotchas

- The component is `SnapshotPage`, not `Snapshot`, because the feature already exports `Snapshot`.
  (`pages/flows` has the same clash and aliases the import instead.)
- `Snapshot` renders `null` until the first state arrives, so the page is an empty padded box for
  that moment. There is no page-level loading state to maintain.
- Nothing here is in the URL: the run's state lives on the server and in the query cache. The
  links out of the page (`/`, `/operations`) belong to the feature's action bars.
