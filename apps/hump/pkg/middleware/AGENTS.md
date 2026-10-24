# pkg/middleware — how this package hangs together

The two pieces of HTTP plumbing the web UI server needs: a CORS middleware driven by an environment
variable, and the handler that serves the embedded single-page application. Its only caller is
`internal/controller/hump_tower.go`, which installs `Cors()` with `router.Use` and `Static` as the
router's `NotFound` handler — so `Static` sees every request the API routes did not claim.

## CORS

`Cors` reads `CORS` **when it is called**, once, as a comma-separated list of origins; blanks are
skipped. With at least one origin it returns a go-chi/cors handler allowing credentials, the usual
methods and `Accept`/`Authorization`/`Content-Type`.

With no usable origin it returns a **pass-through**, so `CORS` is optional: the server then sends
no CORS headers and a browser can call the API from the same origin only. It must never return
nil. chi's `Mux.Use` accepts a nil middleware and panics when it builds the chain, and
`hump_tower.go` calls it inside a function the runner spawned without a `recover`, so a nil here
takes the whole process down at startup.

## Static

Resolution order for a request, first match wins:

1. **A file.** The path names a regular file in the embedded tree. Paths under `static/` get
   `Cache-Control: public, max-age=31536000, immutable` — the build puts content-hashed assets
   there, so anything unhashed must live elsewhere. Directories are never listed.
2. **The shell.** Anything else gets `index.html` with `Cache-Control: no-cache`. The **status** is
   what varies: `200` for `/` and for a path listed in `routes.json`, `404` otherwise. A wrong URL
   still renders the app's own not-found page, but with an honest status code. The shell also
   carries a `Content-Security-Policy`, built once with the handler from `index.html`:
   `script-src 'self'` plus a `sha256-` hash per inline `<script>`, so editing the theme script
   needs no manual update; `style-src 'self' 'unsafe-inline'`, because UI libraries inject styles
   at runtime; everything else `'self'`, with `object-src` and `frame-ancestors` `'none'`. Files
   get no CSP.
3. A plain `404`, only if `index.html` itself is missing.

`routes.json` is the client router's route list, emitted by the web build
(`apps/hump-tower/rsbuild.config.ts`). It is read **once**, when the handler is built, and is never
served itself — a request for it falls through to the shell. A missing or malformed manifest is a
warning, after which every path but `/` answers `404` while still rendering correctly; that is the
symptom to recognise. A client route added without regenerating the manifest behaves the same way.

Trailing slashes are normalised away on both sides, so `/stats/` matches `/stats`.

`Static` takes an `embed.FS` and a sub-directory; if `fs.Sub` fails it logs and returns a handler
that answers `404` to everything rather than failing startup.

## Tests

The static tests build `&static{assets: testAssets()}` over an `fstest.MapFS` and call `handler()`
directly, bypassing `Static` and its `embed.FS` parameter. `TestStaticHandlerStatus` is the table of
which path gets which status. The CORS tests set the variable with `t.Setenv`.
`TestCorsInstallsOnChiWhenUnset` mounts `Cors()` on a real chi mux with `CORS` empty and
registers a route: the regression test for the old nil return.
