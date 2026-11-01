# Contributing to Hump

This guide covers proposing a change, running Hump locally and getting a pull request merged.

- Everyone taking part follows the [Code of Conduct](CODE_OF_CONDUCT.md).
- Report vulnerabilities privately through the [advisory form](https://github.com/hump-cdc/hump/security/advisories/new), never in an issue or pull request. See [SECURITY.md](SECURITY.md).
- Ask questions in [Discussions](https://github.com/hump-cdc/hump/discussions).
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
- Changes that break the layer and import rules in [apps/hump/AGENTS.md](apps/hump/AGENTS.md).
- Anything that weakens at-least-once delivery. Hump confirms a WAL position only after every sink has delivered the change, and that stays.
- New dependencies, Go modules or npm packages, that weren't agreed in the issue.

## Prerequisites

| Tool | Notes | Install |
|---|---|---|
| proto | Installs moon; `moon setup` then installs Go and Bun at the pinned versions | [moonrepo.dev/docs/proto/install](https://moonrepo.dev/docs/proto/install) |
| Docker with Compose v2 | Runs the dev environment (`docker compose`, not `docker-compose`) | [Docker Desktop](https://docs.docker.com/get-docker/), OrbStack or Colima |

You don't install moon, Go or Bun yourself. moon is pinned in [.prototools](.prototools), Go and Bun in [.moon/toolchains.yml](.moon/toolchains.yml).

The Go tools (golangci-lint, gotestsum, gocov, gocov-html) are pinned as `tool` dependencies in [apps/hump/go.mod](apps/hump/go.mod) and run through `go tool`. The first run compiles them, which takes 1–2 minutes. Later runs use the build cache.

On first checkout, review [.prototools](.prototools), then run in the repository root:

```sh
proto trust .prototools   # once; lets proto apply auto-install from .prototools
proto install             # moon, from .prototools
moon setup                # Go and Bun, from .moon/toolchains.yml
bun install               # dependencies of every workspace package
```

proto ignores `auto-install` in an untrusted `.prototools`. Once trusted, the moon shim installs a missing pinned version on first run, for example after the pin changes.

The repository is a [moon](https://moonrepo.dev/) monorepo: the Go service is in `apps/hump`, the web UI in `apps/hump-tower`. Development commands are moon tasks: `moon run <project>:<task>`, from any directory.

## Local development

### Dev environment

```sh
moon run root:dev-up      # docker compose -f dev/docker-compose.yml up -d
moon run root:dev-down    # stop the containers, keep the data
moon run root:dev-reset   # down -v, then up -d: wipes volumes and re-runs dev/compose_init/01-schema.sql
```

The compose file in [dev/docker-compose.yml](dev/docker-compose.yml) starts:

| Service | Host ports | Notes |
|---|---|---|
| `postgres-15` … `postgres-18` | 5435, 5436, **5437**, 5438 | TimescaleDB images, `wal_level=logical`. User, password and database are all `hump`. `moon run hump:run` uses `postgres-17` (5437) |
| `kafka` | 9092 | KRaft single node. Containers reach it at `kafka:29092`. `kafka-init` creates the topics `users`, `posts`, `comments`, `events` |
| `nats` | 4222, monitoring 8222 | JetStream on, user/password `hump`/`hump` |
| `nats2` | 4223, monitoring 8223 | A second, independent NATS server |
| `rabbitmq` | AMQP 5672, management UI 15672, streams 5552 | User/password `hump`/`hump`. `rabbitmq-init` declares the `hump.public` topic exchange and the `hump.public.demo` queue bound with `#` |
| `http-sink` | 5050 | OpenResty target for HTTP sinks, see [HTTP sink routes](#http-sink-routes) |

The schema from [dev/compose_init/01-schema.sql](dev/compose_init/01-schema.sql) creates `users`, `posts`, `comments` (hash-partitioned) and `events` (a hypertable) in every Postgres container.

### Environment

`moon run hump:run` sets these variables (in [apps/hump/moon.yml](apps/hump/moon.yml)). They are fallbacks: a variable already set in your shell wins.

| Variable | Default |
|---|---|
| `DATABASE_URL` | `postgres://hump:hump@localhost:5437/hump?sslmode=disable` |
| `WEB_PORT` | `3200` |
| `TELEMETRY_PORT` | `9100` |
| `KAFKA_HOST` | `localhost:9092` |
| `NATS_URL` | `nats://localhost:4222` |
| `NATS2_URL` | `nats://localhost:4223` |
| `RABBITMQ_URL` | `amqp://hump:hump@localhost:5672/` |
| `RABBITMQ_STREAM_URL` | `rabbitmq-stream://hump:hump@localhost:5552/%2F` |

It also sets `CORS=http://localhost:3000` for the UI dev server, `LOG_FORMAT=pretty` and `LOG_VERBOSE=false`.

To run against another Postgres version, override `DATABASE_URL`:

```sh
DATABASE_URL=postgres://hump:hump@localhost:5435/hump?sslmode=disable moon run hump:run   # PostgreSQL 15
```

### Tasks

The Go service is the `hump` project, the web UI is `hump-tower`, and repo-wide tasks live in the `root` project. Each `moon.yml` has a `description` for every task.

| Task | What it does |
|---|---|
| `moon run hump:run` | Builds the web UI (`hump-tower:build`, which regenerates the OpenAPI spec first), then runs `go run ./cmd/hump -f ../../dev/hump.yaml` in `apps/hump` |
| `moon run hump:run-cleanup` | Removes Hump's replication slot and publication from the dev database |
| `moon run hump:build` | Checks that every package compiles without cgo (`go build ./...` with `CGO_ENABLED=0`), then builds `apps/hump/bin/hump` with the embedded UI |
| `moon run hump:test` | Runs all Go unit tests |
| `moon run hump:test-file` | Runs the tests of the Go package in the current directory |
| `moon run hump:test-cover` | Runs all Go tests with coverage and opens the HTML report |
| `moon run hump:test-file-cover` | The same for the Go package in the current directory |
| `moon run hump:lint` | Runs golangci-lint with the config in [apps/hump/.golangci.yml](apps/hump/.golangci.yml) |
| `moon run hump:lint-fix` | Runs golangci-lint and applies the fixes it can make automatically |
| `moon run hump:fmt` | Formats Go code with gofmt and goimports (`golangci-lint fmt`) |
| `moon run hump:tidy` | Runs `go mod tidy` for `apps/hump`, `apps/hump/tools/actionlint` and `apps/hump/tools/govulncheck` |
| `moon run hump-tower:dev` | UI dev server on port 3000, calling the API at `http://localhost:3200`. Keep `moon run hump:run` going alongside it |
| `moon run hump-tower:storybook` | Storybook on port 6006 |
| `moon run hump-tower:build` | Builds the UI into `apps/hump/internal/controller/hump_tower` |
| `moon run hump-tower:openapi` | Writes the OpenAPI spec to `apps/hump-tower/src/shared/api/openapi.json` |
| `moon run hump-tower:test` | Runs the UI unit tests (`bun test`) |
| `moon run hump-tower:check` | Runs Biome: lint, format, import order |
| `moon run hump-tower:typecheck` | Runs the TypeScript type check |
| `moon run hump-tower:knip` | Finds unused files, exports, types and dependencies |
| `moon run root:actionlint` | Lints the GitHub Actions workflows |
| `moon run root:check` | Runs every check before a pull request, see [Checks](#checks) |

`hump-tower:dev` sets `API_URL=http://localhost:3200` as a fallback. If you change `WEB_PORT`, set `API_URL` to match: `API_URL=http://localhost:3300 moon run hump-tower:dev`.

### Debugging

Use any editor with Delve support. Run it from the repository root: point the debug configuration at the `./apps/hump/cmd/hump` package, pass `-f dev/hump.yaml` as arguments, and set the same environment as `moon run hump:run` (the table above). Containers have to be up first.

## Smoke test

This checks the full path from a row change to the sinks. Run it after any change to a sink or to the WAL path.

1. Start the environment and Hump:

   ```sh
   moon run root:dev-up
   moon run hump:run
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

To test other sinks, comment or uncomment the `- preset: …` lines under `tables:` in `dev/hump.yaml` and restart `moon run hump:run`. The presets are defined under `sinks:` in the same file.

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

The Go service lives in `apps/hump`; the paths below are relative to it. Dependencies point inwards, towards `internal/domain`:

| Layer | Role |
|---|---|
| `internal/domain` | Named types and constants. No behaviour, no I/O |
| `internal/port` | Interfaces and the types they exchange, one package per capability |
| `internal/adapter` | Port implementations: PostgreSQL, sinks, metrics and so on |
| `internal/application` | Use cases grouped by scope (`processing`, `hump_tower`), one package per use case. Depend on ports, never on adapters |
| `internal/controller`, `cmd/hump` | The WAL listener and HTTP servers; `cmd/hump` builds the adapters and injects them |

[apps/hump/AGENTS.md](apps/hump/AGENTS.md) has the full import rules.

### Adding a sink

Open a **New sink** issue first. Once it's approved, the change touches these places (from [apps/hump/internal/adapter/sink/AGENTS.md](apps/hump/internal/adapter/sink/AGENTS.md)), all relative to `apps/hump`:

1. A `SinkType<Name>` constant and an entry in the `SinkTypes` list in `internal/port/sink/sink_type.go`.
2. `config/sink/config_<name>.go` implementing `Definition` (`Type`, `DisplayName`, `Metadata`), plus `CrossValidate` and `UnmarshalYAML`.
3. A case in `unmarshalSink` and an entry in `crossValidators`, both in `config/sink/config_sinks.go`.
4. A case in `internal/adapter/sink/sink.go`, the sink registry.
5. The adapter itself in `internal/adapter/sink/<name>/`, with the same file layout and delivery contract as the existing sinks.

`cmd/hump` already wires the sink registry, so a new sink needs no extra wiring there. A sink that needs a new external capability beyond `Sink` starts with a port in `internal/port`.

### Package notes

Most packages under `apps/hump/internal/` have their own `AGENTS.md`. Read the one for the package you're changing:

- Domain: [domain](apps/hump/internal/domain/AGENTS.md)
- Ports: [database](apps/hump/internal/port/database/AGENTS.md), [live_stream](apps/hump/internal/port/live_stream/AGENTS.md), [metrics_collector](apps/hump/internal/port/metrics_collector/AGENTS.md), [operations_collector](apps/hump/internal/port/operations_collector/AGENTS.md), [sink](apps/hump/internal/port/sink/AGENTS.md), [snapshot](apps/hump/internal/port/snapshot/AGENTS.md)
- Adapters: [live_streamer](apps/hump/internal/adapter/live_streamer/AGENTS.md), [metrics_collector](apps/hump/internal/adapter/metrics_collector/AGENTS.md), [operations_collector](apps/hump/internal/adapter/operations_collector/AGENTS.md), [postgres](apps/hump/internal/adapter/postgres/AGENTS.md), [sink](apps/hump/internal/adapter/sink/AGENTS.md), [snapshot](apps/hump/internal/adapter/snapshot/AGENTS.md)
- Processing use cases: [cleanup_database](apps/hump/internal/application/processing/cleanup_database/AGENTS.md), [process_operation](apps/hump/internal/application/processing/process_operation/AGENTS.md)
- Web UI (hump-tower) use cases: [get_config](apps/hump/internal/application/hump_tower/get_config/AGENTS.md), [get_metrics](apps/hump/internal/application/hump_tower/get_metrics/AGENTS.md), [get_operations](apps/hump/internal/application/hump_tower/get_operations/AGENTS.md), [get_snapshot_estimates](apps/hump/internal/application/hump_tower/get_snapshot_estimates/AGENTS.md), [get_snapshot_tables](apps/hump/internal/application/hump_tower/get_snapshot_tables/AGENTS.md), [start_snapshot](apps/hump/internal/application/hump_tower/start_snapshot/AGENTS.md), [stop_snapshot](apps/hump/internal/application/hump_tower/stop_snapshot/AGENTS.md), [stream_snapshot](apps/hump/internal/application/hump_tower/stream_snapshot/AGENTS.md), [stream_updates](apps/hump/internal/application/hump_tower/stream_updates/AGENTS.md)
- Controllers: [controller](apps/hump/internal/controller/AGENTS.md)

## Generated files

Some generated files are committed. CI fails when the OpenAPI spec or the embedded UI bundle is out of date. Before you open a pull request, regenerate them and commit everything that changed:

- `apps/hump-tower/src/shared/api/openapi.json`, the OpenAPI spec: `moon run hump-tower:openapi`;
- `apps/hump-tower/src/shared/api/generated/`, the API client: `bun run generate:openapi` in `apps/hump-tower` (`hump-tower:build` runs it too);
- `apps/hump/internal/controller/hump_tower/`, the UI bundle embedded into the binary: `moon run hump-tower:build`.

`moon run hump:build` runs all three.

## Tests

- New behaviour comes with unit tests.
- A bug fix comes with a test that fails without the fix.
- Coverage of the packages you touch must not drop. CI checks coverage through Codecov: Go or web UI total coverage may not drop by more than 1 point, and changed lines need 80% coverage. Locally, `moon run hump:test-cover` opens an HTML report for the whole Go module, and `moon run hump:test-file-cover` does the same for the package in the current directory.
- A change to a sink or to the WAL path also needs the [smoke test](#smoke-test). Describe what you ran in the **Testing** section of the pull request.

## Checks

`moon run root:check` has to pass. It runs actionlint, `hump:lint`, `hump:test` and, for the web UI, `hump-tower:check` (Biome), `hump-tower:test`, `hump-tower:typecheck` and `hump-tower:knip`.

CI ([.github/workflows/ci-check.yml](.github/workflows/ci-check.yml)) runs the full set of checks on every pull request in three parallel jobs, one per moon project (Go app, web UI, repository-wide). Explicit steps call the same moon commands used locally, plus security and dependency-policy checks. Independent checks continue after a failure, but the job still fails. CI also fails when `go.mod`/`go.sum` are not tidy, when the OpenAPI spec or the embedded UI bundle is stale, or when `bun.lock` is out of sync with the `package.json` files.

[.github/workflows/main-security-scan.yml](.github/workflows/main-security-scan.yml) runs on every push to `main`, every Monday and on manual dispatch. It scans the workflows with zizmor and the Go module `apps/hump` with govulncheck, and uploads both reports to the repository's **Security** tab. It also audits direct and transitive npm packages in the root `bun.lock` and separately checks that each one comes from registry.npmjs.org. Security findings appear in the job log and Summary, with machine-readable reports available as artifacts. CI passes its audit exclusions from Actions; `moon run hump-tower:audit` checks all packages by default. The weekly runs also catch vulnerabilities published after the last change to `apps/hump` or `bun.lock`.

[.github/workflows/main-scorecard.yml](.github/workflows/main-scorecard.yml) runs the [OpenSSF Scorecard](https://scorecard.dev/) checks on every push to `main`, every Thursday and on manual dispatch. It uploads the findings to the **Security** tab and publishes the score shown by the badge in the README.

Every job in every workflow starts with [Harden-Runner](https://github.com/step-security/harden-runner) in audit mode, which records the outbound network connections of the job. Once the audit runs show which endpoints each job needs, it switches to block mode, and a connection to any other endpoint fails the job.

Dependabot proposes a new dependency version 7 days after its release, and a new major version of a Go module or npm package after 30 days. Minor and patch updates of each Go module and of the Bun workspace come in one grouped pull request, every major update in its own. All GitHub Actions updates come in one pull request.

## Commits and pull requests

- Write the commit message as one English sentence in the imperative mood: "Add RabbitMQ Streams sink", "Fix retry backoff for HTTP 429". Don't use Conventional Commits prefixes such as `feat:` or `fix:`.
- Pull requests are squash-merged, and the PR title becomes the commit message, so the same rule applies to the title.
- Fill in the [pull request template](.github/pull_request_template.md), including **Breaking changes** and **Testing**.
- Link the issue: `Closes #123`.
- Leave **Allow edits from maintainers** on.
- Contributions are licensed under the [MIT License](LICENSE). There is no CLA or DCO.

## AI-assisted contributions

AI tools are welcome. Agent instructions are in [AGENTS.md](AGENTS.md), [apps/hump/AGENTS.md](apps/hump/AGENTS.md), [apps/hump-tower/AGENTS.md](apps/hump-tower/AGENTS.md) and the per-package `AGENTS.md` files, so agents get the same rules as people.

You are the author, though. You need to understand and have reviewed every line you submit. Generated issues and pull requests that nobody reviewed are closed.
