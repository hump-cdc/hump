# application/hump_tower/get_snapshot_tables — how this package hangs together

`POST /api/snapshot/tables` (`snapshot.tables`; no request body). It fills the snapshot dialog:
every configured table as the database currently describes it — type, columns, a suggested filter
— plus the sinks a snapshot of it would reach.

## Where each field comes from

- **Type, columns, `where`** come from the database port in one `GetTables` call. The suggested
  filter is picked by `adapter/postgres/snapshot_where_suggester.go`; nothing here inspects
  columns or builds SQL.
- **Sinks** come from the configuration, joined by `domain.TableIdentifier`.

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
- `tableSinks` must render like every other view (see `port/sink/AGENTS.md`). A nil `Sink` is
  skipped.
- A nil database answers an **empty list**, not an error.
- `columns` and `sinks` always allocate, so the client gets `[]`, never `null`. The generated zod
  schema marks both `.nullable()`, so a `null` would still pass validation; the code does not say
  why it allocates.
- The `type` enum tag in `dto.go` repeats the port's `TableType` values by hand.
