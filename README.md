<div align="center">

<img src="apps/hump-tower/public/logo_lg.svg" width="120" alt="Hump logo">

<h1>Hump</h1>

<p>Your database, as an event stream.</p>

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/hump-cdc/hump)](https://github.com/hump-cdc/hump/releases)
![PostgreSQL 15+](https://img.shields.io/badge/PostgreSQL-15%2B-336791)

[Website](https://hump.dev) · [Documentation](https://hump.dev/docs/getting-started)

</div>

In a microservice system each service owns its database, and other services still need to know what changed in it. Hump reads every insert, update, delete and truncate from PostgreSQL logical replication and publishes each change as an event.

It uses the built-in `pgoutput` plugin, so PostgreSQL needs no extensions. Your service keeps writing to its own tables. You don't add event-publishing code, an outbox table or dual writes.

- One binary of about 30 MB that uses 20–30 MB of RAM. You run one instance next to each database and start it with one command.
- Hump creates its publication and replication slot on start. `--cleanup` drops them.
- Events use a Debezium-compatible envelope by default. Every topic, subject, routing key, header and payload is a Go template.
- Delivery is at-least-once. Hump confirms the WAL position on transaction commit, after every sink has delivered. Each sink has its own retry policy (exponential or linear backoff) and failure policy (`crash` or `skip`).
- Snapshots backfill existing rows into the same sinks.
- A built-in web console shows live events and stats and runs snapshots. Prometheus metrics and health probes are included.
- MIT-licensed, with no telemetry.

## Supported

| | |
|---|---|
| Source | PostgreSQL 15+, including declarative partitions and TimescaleDB hypertables |
| Sinks | Kafka (TLS, SASL), NATS Core, NATS JetStream, RabbitMQ (AMQP 0.9.1), RabbitMQ Streams, HTTP webhooks |

## Quick start

1. Prepare PostgreSQL. Enable logical replication and restart the server:

   ```sql
   ALTER SYSTEM SET wal_level = logical;
   ```

   `max_replication_slots` and `max_wal_senders` must be at least 1; the defaults are fine. The user Hump connects as needs the `REPLICATION` attribute or superuser rights. Hump creates the publication and the replication slot itself.

2. Install. Download the binary for your platform from [GitHub Releases](https://github.com/hump-cdc/hump/releases).

3. Configure. Create `hump.yaml`:

   ```yaml
   tables:
     - schema: public
       table: users
       sinks:
         - kafka:
             url: "localhost:9092"
             topic: "{{ .Schema }}.{{ .Table }}"
   ```

4. Run:

   ```sh
   hump --database-url "postgres://user:password@localhost:5432/app" --config-file hump.yaml
   ```

Every change to `public.users` now lands in the `public.users` topic. See the [documentation](https://hump.dev/docs/getting-started) for all options.

## Documentation

The full documentation is at [hump.dev/docs](https://hump.dev/docs/getting-started). It covers configuration, sinks, templating, delivery guarantees, snapshots and observability.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to propose changes, set up the dev environment and open a pull request.

## License

Hump is released under the [MIT License](LICENSE).
