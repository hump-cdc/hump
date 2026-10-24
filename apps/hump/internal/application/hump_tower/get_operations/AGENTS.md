# application/hump_tower/get_operations — how this package hangs together

`POST /api/operations` (`operations.list`). One page of the operations log: deliveries of one row
to one sink, newest first, cut by a filter. The log itself — the ring, the filter, the paging — is
`adapter/operations_collector`; this package turns a request into a `Query` and a `Page` into JSON.

## The filter is not applied here, and must not be

`Search` hands the whole filter to `collector.Search`, which matches inside the ring and clones
only the records of the requested window. Filtering in this package instead would mean dumping the
ring — `capacity × payloadLimit`, every buffer deep-copied — to throw most of it away. See
`adapter/operations_collector/AGENTS.md`, "The filter runs inside the ring".

What that implies for the request:

- `from` / `to` are **indices into the matches**, counted from the newest, half-open: `from: 0,
  to: 50` is the 50 newest matching records. They are not times and not record numbers.
- The ring clamps both to `Total`, so a window past the end is an empty page, not an error. The
  response's `from` / `to` are the **clamped** values (`Page.Start` / `Page.End`), not an echo of
  the request — `TestHandlerReportsTheClampedWindow`.
- `targets` matches the **configured** target (`Record.TargetKey`, what the metrics topology is
  keyed by), while the `target` of a returned item is the one **rendered for that row**. A
  templated topic makes the two differ; that is what lets the flows inspector list the deliveries
  of one node.
- Items stay in the order the collector answered in. Nothing here re-sorts.

`ErrInvertedWindow` (`to < from`, 422) is the single validation this package adds; `minimum:"0"`
on both fields is huma's.

## `convert` keeps an empty list nil

`convert` casts `[]string` to the domain vocabulary and returns **nil** for an empty input. In the
ring an empty list means "match everything", so an omitted filter and an empty one behave the same.
The values are cast, not validated: the `enum` tags on `operations` and `statuses` are what keeps a
stray value out. `statuses` lists only `success` and `failure` on purpose — `retry` is an attempt's
status and never a record's.

## Nil versus idle collector

A nil collector is `ErrDisabled` (503); `cmd/hump/main.go` builds one whenever the web port is
set. An **idle** collector is a different thing — it answers
an empty page. Only `/api/live` calls `Connect` (see `stream_updates`), so records accumulate while
a client holds that stream open and until the collector's idle timeout after it leaves; a bare call
to this route never turns the log on.

## Rendering

- Durations are whole **milliseconds** via `toMillis`, the unit every latency on the API is
  carried in, with no unit in the field name.
- `attempts[].attempt` is numbered from 1 here, so the UI never derives it from a slice position.
- `metadata` substitutes `{}` for a nil map: the field is required, and a JSON `null` fails the
  generated client's schema. The map returned is otherwise the page's own — safe, because `Search`
  deep-copies every record.
- `payload` is the body truncated to the collector's limit; `payloadSize` is the untruncated
  length.

## Tests

`controller_test.go` covers the use case through the handler; there is no `usecase_test.go`.
