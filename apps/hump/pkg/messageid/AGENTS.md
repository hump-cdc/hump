# pkg/messageid — how this package hangs together

`Hash(parts ...string)` in `hash.go` hashes ordered source-event identity components. It uses
only the standard library; keep it reusable, with no imports from `internal` or `config`.

## Encoding contract

Each component is its byte length as an unsigned 64-bit big-endian integer, followed by its
unmodified bytes. SHA-256 hashes that sequence; the result is 64 lowercase hexadecimal characters.
Component boundaries, empty strings and embedded NUL bytes remain distinct. Do not replace the
length prefixes with delimiter joining or normalize inputs.

Changing the encoding or component order changes existing message ids. Preserve both when
refactoring. Keep the exported function before any helpers.

## Callers own the identity

`Hash` does not choose or validate components. The adapters supply these ordered tuples:

- `internal/adapter/postgres/wal.go`: `"WAL"`, database, configured schema, configured table,
  `Begin.FinalLSN.String()`, decimal change ordinal. The ordinal counts INSERT, UPDATE, DELETE
  and TRUNCATE messages before table filtering and resets between transactions and sessions.
- `internal/adapter/snapshot/snapshot.go`: `"snapshot"`, database, configured schema, configured
  table, exported snapshot id, physical leaf schema, physical leaf table, CTID. A new exported
  snapshot produces different row identities.

The case-sensitive source markers separate WAL and snapshot identities. Keep physical snapshot
row metadata separate from user record fields; this package hashes strings, not payloads.

## Tests

`TestHashSeparatesComponents` checks repeatability, digest length and separation of ambiguous
concatenations, empty components, NUL bytes and source markers. Extend it when changing encoding.
