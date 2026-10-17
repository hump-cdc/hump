# Contributing to Hump

This guide covers proposing a change, running Hump locally and getting a pull request merged.

- Everyone taking part follows the [Code of Conduct](CODE_OF_CONDUCT.md).
- Report vulnerabilities privately through the [advisory form](https://github.com/sergeyzwezdin/hump/security/advisories/new), never in an issue or pull request. See [SECURITY.md](SECURITY.md).
- Ask questions in [Discussions](https://github.com/sergeyzwezdin/hump/discussions).
- User documentation lives at [hump.dev/docs](https://hump.dev/docs/getting-started).

## Open an issue first

Every pull request needs an issue that the maintainer has approved. There are no exceptions, including small fixes. A pull request without an approved issue is closed.

Pick the template that fits:

- **Bug report**: something doesn't work as documented.
- **Feature request**: new behaviour or an option.
- **New sink**: a new destination for change events.

Blank issues are disabled.

## What we don't accept

- Cosmetic refactors: renames, reformatting or moving code around with no functional purpose.
- Changes that break the layer and import rules in [AGENTS.md](AGENTS.md).
- Anything that weakens at-least-once delivery. Hump confirms a WAL position only after every sink has delivered the change, and that stays.
- New dependencies, Go modules or npm packages, that weren't agreed in the issue.

## Prerequisites

| Tool | Notes | Install |
|---|---|---|
| Go | Version from the `go` directive in [go.mod](go.mod) (1.26.1 at the time of writing) | [go.dev/dl](https://go.dev/dl/) or `brew install go` |
| Docker with Compose v2 | Runs the dev environment (`docker compose`, not `docker-compose`) | [Docker Desktop](https://docs.docker.com/get-docker/), OrbStack or Colima |
| Task | Runs everything in [taskfile.yaml](taskfile.yaml) | [taskfile.dev](https://taskfile.dev/installation/) or `brew install go-task` |
| bun | Builds and tests the web UI | [bun.sh](https://bun.sh/) or `brew install oven-sh/bun/bun` |

The Go tools (golangci-lint, gotestsum, gocov, gocov-html) are pinned as `tool` dependencies in [go.mod](go.mod) and run through `go tool`, so there is nothing to install. The first run compiles them, which takes 1–2 minutes. Later runs use the build cache.

On first checkout, install the web UI dependencies:

```sh
cd web/webui && bun install
```

## Local development

### Dev environment

```sh
task dev:up      # docker compose -f dev/docker-compose.yml up -d
task dev:down    # stop the containers, keep the data
task dev:reset   # down -v, then up -d: wipes volumes and re-runs dev/compose_init/01-schema.sql
```

The compose file in [dev/docker-compose.yml](dev/docker-compose.yml) starts:

| Service | Host ports | Notes |
|---|---|---|
| `postgres-15` … `postgres-18` | 5435, 5436, **5437**, 5438 | TimescaleDB images, `wal_level=logical`. User, password and database are all `hump`. `task run` uses `postgres-17` (5437) |
| `kafka` | 9092 | KRaft single node. Containers reach it at `kafka:29092`. `kafka-init` creates the topics `users`, `posts`, `comments`, `events` |
| `nats` | 4222, monitoring 8222 | JetStream on, user/password `hump`/`hump` |
| `nats2` | 4223, monitoring 8223 | A second, independent NATS server |
| `rabbitmq` | AMQP 5672, management UI 15672, streams 5552 | User/password `hump`/`hump`. `rabbitmq-init` declares the `hump.public` topic exchange and the `hump.public.demo` queue bound with `#` |
| `http-sink` | 5050 | OpenResty target for HTTP sinks, see [HTTP sink routes](#http-sink-routes) |

The schema from [dev/compose_init/01-schema.sql](dev/compose_init/01-schema.sql) creates `users`, `posts`, `comments` (hash-partitioned) and `events` (a hypertable) in every Postgres container.

### Environment

`task run` sets these variables:

| Variable | Default | Override from your shell |
|---|---|---|
| `DATABASE_URL` | `postgres://hump:hump@localhost:5437/hump?sslmode=disable` | Yes |
| `WEB_PORT` | `3200` | Yes |
| `TELEMETRY_PORT` | `9100` | Yes |
| `KAFKA_HOST` | `localhost:9092` | No, edit `taskfile.yaml` |
| `NATS_URL` | `nats://localhost:4222` | No |
| `NATS2_URL` | `nats://localhost:4223` | No |
| `RABBITMQ_URL` | `amqp://hump:hump@localhost:5672/` | No |
| `RABBITMQ_STREAM_URL` | `rabbitmq-stream://hump:hump@localhost:5552/%2F` | No |

It also sets `CORS=http://localhost:3000` for the UI dev server, `LOG_FORMAT=pretty` and `LOG_VERBOSE=false`.

To run against another Postgres version, override `DATABASE_URL`:

```sh
DATABASE_URL=postgres://hump:hump@localhost:5435/hump?sslmode=disable task run   # PostgreSQL 15
```

### Tasks

| Task | What it does |
|---|---|
| `task run` | Builds the web UI (`task web:build`, which regenerates the OpenAPI spec first), then runs `go run ./cmd/hump -f dev/hump.yaml` |
| `task run:cleanup` | Removes Hump's replication slot and publication from the dev database |
| `task web:dev` | UI dev server on port 3000, calling the API at `http://localhost:3200`. Keep `task run` going alongside it |
| `task web:storybook` | Storybook on port 6006 |
| `task build` | Builds `./bin/hump` with the embedded UI |
| `task openapi` | Writes the OpenAPI spec to `web/webui/src/shared/api/openapi.json` |
| `task lint` | Runs golangci-lint with the config in [.golangci.yml](.golangci.yml) |
| `task lint:fix` | Runs golangci-lint and applies the fixes it can make automatically |
| `task fmt` | Formats Go code with gofmt and goimports (`golangci-lint fmt`) |

`task web:dev` has the API URL fixed in `taskfile.yaml`, so it won't follow a changed `WEB_PORT`.

### Debugging

Use any editor with Delve support. Point the debug configuration at the `./cmd/hump` package, pass `-f dev/hump.yaml` as arguments, and set the same environment as `task run` (the table above). Containers have to be up first.

## Smoke test

This checks the full path from a row change to the sinks. Run it after any change to a sink or to the WAL path.

1. Start the environment and Hump:

   ```sh
   task dev:up
   task run
   ```

   Start Hump before writing rows. It creates the replication slot on its first start, and changes made before that aren't captured.

2. Open `psql` in the default Postgres container:

   ```sh
   docker compose -f dev/docker-compose.yml exec postgres-17 psql -U hump
   ```

3. Make some changes:

   ```sql
   INSERT INTO users (email, name) VALUES ('ada@example.com', 'Ada');

   INSERT INTO posts (author_id, title, body)
   VALUES ((SELECT id FROM users WHERE email = 'ada@example.com'), 'Hello', 'First post');

   UPDATE posts SET title = 'Hello again', version = version + 1 WHERE title = 'Hello';

   DELETE FROM posts WHERE title = 'Hello again';
   ```

4. Check where the events went. With [dev/hump.yaml](dev/hump.yaml) as committed:

   - `users` routes inserts only, to `kafka1`.
   - `posts`, `comments` and `events` route inserts, updates and deletes to `kafka1` and `nats-jetstream`.

   Kafka (`kafka1` writes to a topic named after the table). Run the consumer inside the container on the internal listener:

   ```sh
   docker compose -f dev/docker-compose.yml exec kafka \
     /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server kafka:29092 \
     --topic posts --from-beginning
   ```

   You should see one insert, one update and one delete for `posts`, and one insert on the `users` topic.

   NATS JetStream (`nats-jetstream` publishes to `hump.js.public.<table>` in the `HUMP` stream on `nats`). Check the message count at `http://localhost:8222/jsz?streams=true`. If you have the optional [nats CLI](https://github.com/nats-io/natscli), you can read the messages too:

   ```sh
   nats --server nats://hump:hump@localhost:4222 stream view HUMP
   ```

5. Open the web UI at `http://localhost:3200` (`WEB_PORT`) and check the live events and stats. Prometheus metrics are at `http://localhost:9100/metrics` (`TELEMETRY_PORT`), and the health probes are at `/health/startup`, `/health/ready` and `/health/live` on the same port.

To test other sinks, comment or uncomment the `- preset: …` lines under `tables:` in `dev/hump.yaml` and restart `task run`. The presets are defined under `sinks:` in the same file.

### HTTP sink routes

The `http-sink` container on port 5050 serves four routes, defined in [dev/compose_init/http-sink.conf](dev/compose_init/http-sink.conf):

| Route | Behaviour | Preset in `dev/hump.yaml` |
|---|---|---|
| `/submit1` | Waits 50 ms, returns 200 | `http1` |
| `/submit2` | Logs the body to the container output, waits 1.5 s, then returns 500 for about 95% of requests and 200 for the rest | `http2` |
| `/submit3` | Waits 75 ms, returns 500 | none |
| `/submit4` | Logs a line, waits 15 s, logs the body, returns 200 | `slow-http` |

Read the logged bodies with `docker compose -f dev/docker-compose.yml logs -f http-sink`.

## Architecture

Dependencies point inwards, towards `internal/domain`:

| Layer | Role |
|---|---|
| `internal/domain` | Named types and constants. No behaviour, no I/O |
| `internal/port` | Interfaces and the types they exchange, one package per capability |
| `internal/adapter` | Port implementations: PostgreSQL, sinks, metrics and so on |
| `internal/application` | Use cases grouped by scope (`processing`, `webui`), one package per use case. Depend on ports, never on adapters |
| `internal/controller`, `cmd/hump` | The WAL listener and HTTP servers; `cmd/hump` builds the adapters and injects them |

[AGENTS.md](AGENTS.md) has the full import rules.

### Adding a sink

Open a **New sink** issue first. Once it's approved, the change touches these places (from [internal/adapter/sink/AGENTS.md](internal/adapter/sink/AGENTS.md)):

1. A `SinkType<Name>` constant and an entry in the `SinkTypes` list in `internal/port/sink/sink_type.go`.
2. `config/sink/config_<name>.go` implementing `Definition` (`Type`, `DisplayName`, `Metadata`), plus `CrossValidate` and `UnmarshalYAML`.
3. A case in `unmarshalSink` and an entry in `crossValidators`, both in `config/sink/config_sinks.go`.
4. A case in `internal/adapter/sink/sink.go`, the sink registry.
5. The adapter itself in `internal/adapter/sink/<name>/`, with the same file layout and delivery contract as the existing sinks.

`cmd/hump` already wires the sink registry, so a new sink needs no extra wiring there. A sink that needs a new external capability beyond `Sink` starts with a port in `internal/port`.

### Package notes

Most packages under `internal/` have their own `AGENTS.md`. Read the one for the package you're changing:

- Domain: [domain](internal/domain/AGENTS.md)
- Ports: [database](internal/port/database/AGENTS.md), [live_stream](internal/port/live_stream/AGENTS.md), [metrics_collector](internal/port/metrics_collector/AGENTS.md), [operations_collector](internal/port/operations_collector/AGENTS.md), [sink](internal/port/sink/AGENTS.md), [snapshot](internal/port/snapshot/AGENTS.md)
- Adapters: [live_streamer](internal/adapter/live_streamer/AGENTS.md), [metrics_collector](internal/adapter/metrics_collector/AGENTS.md), [operations_collector](internal/adapter/operations_collector/AGENTS.md), [postgres](internal/adapter/postgres/AGENTS.md), [sink](internal/adapter/sink/AGENTS.md), [snapshot](internal/adapter/snapshot/AGENTS.md)
- Processing use cases: [cleanup_database](internal/application/processing/cleanup_database/AGENTS.md), [process_operation](internal/application/processing/process_operation/AGENTS.md)
- Web UI use cases: [get_config](internal/application/webui/get_config/AGENTS.md), [get_metrics](internal/application/webui/get_metrics/AGENTS.md), [get_operations](internal/application/webui/get_operations/AGENTS.md), [get_snapshot_estimates](internal/application/webui/get_snapshot_estimates/AGENTS.md), [get_snapshot_tables](internal/application/webui/get_snapshot_tables/AGENTS.md), [start_snapshot](internal/application/webui/start_snapshot/AGENTS.md), [stop_snapshot](internal/application/webui/stop_snapshot/AGENTS.md), [stream_snapshot](internal/application/webui/stream_snapshot/AGENTS.md), [stream_updates](internal/application/webui/stream_updates/AGENTS.md)
- Controllers: [controller](internal/controller/AGENTS.md)

## Generated files

Some generated files are committed. Before you open a pull request, run `task build` and commit everything it changed:

- `web/webui/src/shared/api/openapi.json`, the OpenAPI spec;
- `web/webui/src/shared/api/generated/`, the API client;
- `internal/controller/webui/`, the UI bundle embedded into the binary.

## Tests

- New behaviour comes with unit tests.
- A bug fix comes with a test that fails without the fix.
- Coverage of the packages you touch must not drop. CI checks this. Locally, `task test:cover` opens an HTML report for the whole module, and `task test:file:cover` does the same for the package in the current directory.
- A change to a sink or to the WAL path also needs the [smoke test](#smoke-test). Describe what you ran in the **Testing** section of the pull request.

## Checks

`task check` has to pass. It runs `task lint` and `task test`, then, in `web/webui`, `biome check`, `bun test` and `tsc --noEmit`. CI runs the same checks on every pull request.

## Commits and pull requests

- Write the commit message as one English sentence in the imperative mood: "Add RabbitMQ Streams sink", "Fix retry backoff for HTTP 429". Don't use Conventional Commits prefixes such as `feat:` or `fix:`.
- Pull requests are squash-merged, and the PR title becomes the commit message, so the same rule applies to the title.
- Fill in the [pull request template](.github/pull_request_template.md), including **Breaking changes** and **Testing**.
- Link the issue: `Closes #123`.
- Leave **Allow edits from maintainers** on.
- Contributions are licensed under the [MIT License](LICENSE). There is no CLA or DCO.

## AI-assisted contributions

AI tools are welcome. Agent instructions are in [AGENTS.md](AGENTS.md) and the per-package `AGENTS.md` files, so agents get the same rules as people.

You are the author, though. You need to understand and have reviewed every line you submit. Generated issues and pull requests that nobody reviewed are closed.
