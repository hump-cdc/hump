# pages/not-found — how this page hangs together

The 404 screen. It has **no route file**: `app/router.tsx` passes it to the router as
`defaultNotFoundComponent`, so it is the fallback for any URL that matches no route.

## Layout

| File | Role |
|---|---|
| `index.ts` | public contract: `NotFound` |
| `ui/not-found.tsx` | the fake terminal session and the way back |

## What it is made of

Only `@/shared/ui/error` (`ErrorTerminal`, `ErrorPrompt`, `ErrorLine`) and `@/shared/ui/button` —
no entity, no query, no store. `ErrorTerminal` is a full-screen (`min-h-screen`) layout of its own.

The only dynamic part is the path echoed in the first two terminal lines. It comes from
`useLocation().pathname`, is cut to 48 characters with a trailing `…` so it cannot break the
terminal frame, and falls back to `/the-page-you-wanted` when the pathname is empty or `/`.

The action is a `Link` to `/` with `replace`, so the dead URL does not stay in history. Its label,
"Go to Flows", is true only while `app/routes/_sidebar/index.tsx` mounts the Flows page.
