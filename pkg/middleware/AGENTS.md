# pkg/middleware — how this package hangs together

The two pieces of HTTP plumbing the web UI server needs: a CORS middleware driven by an environment
variable, and the handler that serves the embedded single-page application. Its only caller is
`internal/controller/webui.go`, which installs `Cors()` with `router.Use` and `Static` as the
router's `NotFound` handler — so `Static` sees every request the API routes did not claim.

## Layout

| File | Role |
|---|---|
| `cors.go` | `Cors` — go-chi/cors configured from the `CORS` environment variable |
| `static.go` | `Static` — assets, the SPA shell fallback, the route manifest |

## CORS

`Cors` reads `CORS` **when it is called**, once, as a comma-separated list of origins; blanks are
skipped. With at least one origin it returns a go-chi/cors handler allowing credentials, the usual
methods and `Accept`/`Authorization`/`Content-Type`.

With no usable origin it returns **nil**, and chi does not tolerate that: `Mux.Use` appends the nil
function and the chain panics with a nil dereference as soon as it is built. `webui.go` passes the
result to `router.Use` unguarded, inside a function the runner spawned without a `recover`, so
**hump panics at startup when the web port is set and `CORS` is empty** — the whole process, not
only the web server. `task run` sets `CORS`, which is why it does not show up there. Guard the call
site or return a pass-through before relying on `CORS` being optional.

## Static

Resolution order for a request, first match wins:

1. **A file.** The path names a regular file in the embedded tree. Paths under `static/` get
   `Cache-Control: public, max-age=31536000, immutable` — the build puts content-hashed assets
   there, so anything unhashed must live elsewhere. Directories are never listed.
2. **The shell.** Anything else gets `index.html` with `Cache-Control: no-cache`. The **status** is
   what varies: `200` for `/` and for a path listed in `routes.json`, `404` otherwise. A wrong URL
   still renders the app's own not-found page, but with an honest status code.
3. A plain `404`, only if `index.html` itself is missing.

`routes.json` is the client router's route list, emitted by the web build
(`web/webui/rsbuild.config.ts`). It is read **once**, when the handler is built, and is never served
itself — a request for it falls through to the shell. A missing or malformed manifest is a warning,
after which every path but `/` answers `404` while still rendering correctly; that is the symptom to
recognise. A client route added without regenerating the manifest behaves the same way.

Trailing slashes are normalised away on both sides, so `/stats/` matches `/stats`.

`Static` takes an `embed.FS` and a sub-directory; if `fs.Sub` fails it logs and returns a handler
that answers `404` to everything rather than failing startup.

## Tests

The static tests build `&static{assets: testAssets()}` over an `fstest.MapFS` and call `handler()`
directly, bypassing `Static` and its `embed.FS` parameter. `TestStaticHandlerStatus` is the table of
which path gets which status. The CORS tests set the variable with `t.Setenv`.
