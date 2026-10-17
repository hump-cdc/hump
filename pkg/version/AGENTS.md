# pkg/version — how this package hangs together

Three string variables the linker fills in. `version.go` declares `Version`, `Commit` and `Date`;
`taskfile.yaml` sets them through `LDFLAGS`:

```
-X hump/pkg/version.Version=… -X hump/pkg/version.Commit=… -X hump/pkg/version.Date=…
```

## What to know

- **The path in `-X` is the coupling.** Renaming a variable or moving the package does not break the
  build — the linker ignores an `-X` that matches nothing — it just leaves every binary reporting
  the defaults. Change `taskfile.yaml` in the same edit.
- A plain `go build` or `go test` gets the defaults: `Version` is `development`, `Commit` and `Date`
  are empty. Code that reads them must be fine with that; `log.Scope` omits `commit` when it is
  empty.
- `Version` reaches the log attributes, the health component, the OpenAPI document, the HTTP sink's
  `User-Agent`, the RabbitMQ client properties, the web UI config and, per row,
  `sink.Context.Version` (`process_operation/usecase.go`), which sink templates render as
  `.Version` (the default payload's `source.version`). `Commit` reaches the logs and
  the web UI config. `Date` is set by the build and read by nothing.
- They must stay `var` of type `string` with a constant initialiser; `-X` cannot set anything else.
