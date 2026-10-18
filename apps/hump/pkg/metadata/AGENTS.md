# pkg/metadata — how this package hangs together

One function. `ParseHost` reduces a rendered sink connection string to the endpoint that identifies
it, and every per-sink-type file (`config/sink/config_http.go`, `config_kafka.go`,
`config_nats_core.go`, `config_nats_jetstream.go`, `config_rabbitmq_classic.go`,
`config_rabbitmq_stream.go`) calls it from `Definition.Metadata` to produce the `host` half of the
`(host, target)` pair.

## Why it matters more than its size

That pair becomes the `host` and `target` Prometheus labels on the sink metrics, and the dashboard
resolves its topology with the same `Metadata` call so the two sides match **byte for byte** — see
`adapter/metrics_collector/AGENTS.md`, "Topology comes from outside". Changing what `ParseHost`
returns for an existing input renames live metric series and, if only one side moves, empties the
dashboard without an error anywhere.

## What it returns

- A URL with a scheme and an authority (`nats://`, `amqp://`, `https://`, `rabbitmq-stream://`)
  yields `url.URL.Host`: scheme, credentials, path and query are dropped, the port is kept.
- Anything `net/url` does not see a host in is returned **verbatim** after trimming whitespace. That
  is the Kafka case — a bare `host:port` or a comma-separated broker list — and it is a fallback,
  not a parse: a string of that shape with credentials in it would be returned with them.
- An empty or blank string yields `""`.

## Tests

`TestParseHost` is the table of accepted shapes; add a row there for any new sink URL format.
