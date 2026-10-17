# application/webui/get_snapshot_tables — how this package hangs together

`POST /api/snapshot/tables` (`snapshot.tables`, registered in `internal/controller/webui.go`; no
request body). It fills the snapshot dialog: every configured table as the database currently
describes it — type, columns, a suggested filter — plus the sinks a snapshot of it would reach.

## Layout

| File | Role |
|---|---|
| `usecase.go` | `Tables`: configured identifiers → `db.GetTables` → descriptions joined with their sinks |
| `controller.go` | the huma handler and the column and sink renderers |
| `dto.go` | `SnapshotTablesResponse` and its parts — the wire contract |

## Where each field comes from

- **Type, columns, `where`** come from the database port in one `GetTables` call. The suggested
  filter is picked by `adapter/postgres/snapshot_where_suggester.go`; nothing here inspects
  columns or builds SQL.
- **Sinks** come from the configuration, joined by `domain.TableIdentifier`. `Table` embeds
  `database.TableDescription` and adds them.

The result follows the port's order, and the port is asked in configuration order —
`TestTablesAsksForEveryConfiguredTableInOrder` pins the asking side.

## Things that will bite

- **A configured table missing from the database is silently absent.** `GetTables` skips it, so
  the response can be shorter than the configuration. That is not an error here.
- **Every database failure is a 422**, a connection problem included; the handler does not
  classify.
- **Sinks are not filtered by operation.** `tableSinks` lists each distinct `(type, host, target)`
  of the table, whereas a run — `adapter/snapshot` — lists and counts only sinks whose configured
  operations include `insert`. A table with an update-only sink shows that sink here and not in
  the run's progress.
- `tableSinks` renders with `Metadata(SinkContext{Schema, Table})`, the same call the metrics
  topology and `get_config.tableTargets` make, so the strings line up across views. A nil `Sink`
  is skipped.
- A nil database answers an **empty list**, not an error. That path exists for `cmd/openapi`,
  which builds the use case with nil to register the route, and for tests.
- `columns` and `sinks` always allocate, so the client gets `[]`, never `null`. The generated zod
  schema marks both `.nullable()`, so a `null` would still pass validation; the code does not say
  why it allocates.

`dto.go` tags are emitted into `web/webui/src/shared/api/openapi.json` by `task openapi`. The
`type` enum repeats the port's `TableType` values by hand.

## Tests

`usecase_test.go` covers the join and the nil database; `controller_test.go` covers rendering and
the 422. Both share one package: `fakeDatabase`, `fakeSink`, `configWith` and `identifier` are
defined once, in `usecase_test.go`. Helpers live below the last test function — see the `go` skill.
