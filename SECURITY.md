# Security policy

## Reporting a vulnerability

Report vulnerabilities privately through GitHub: open the **Security** tab and click **Report a vulnerability**, or go straight to the [advisory form](https://github.com/hump-cdc/hump/security/advisories/new). Don't open a public issue, pull request or discussion for a vulnerability.

A useful report includes:

- the Hump version or commit (every log line carries `version`, and `commit` when the build sets it);
- the relevant part of the configuration, with secrets removed;
- the PostgreSQL version and the version of the sink involved;
- steps to reproduce or a proof of concept;
- the impact: what an attacker can read, change or break.

## Supported versions

| Version | Supported |
|---|---|
| Latest release | Yes |
| `main` | Yes |
| Older releases | No |

Fixes land in `main` and ship in the next release. They are not backported. Compatibility between releases is described in [VERSIONING.md](VERSIONING.md).

## What happens next

Hump has one maintainer, so security work is best effort. You should get an acknowledgement within about 7 days. There are no guaranteed timelines after that.

A confirmed vulnerability is assessed, fixed in `main` and shipped in a release. Then a GitHub Security Advisory is published, with a CVE when one applies. Reporters are credited unless they ask not to be.

Please keep the details private until the fixed release and the advisory are out.

## Scope

Out of scope:

- Vulnerabilities in dependencies. Report them upstream. If Hump uses the affected code in an exploitable way, a report here is welcome too.
- Issues that need an already compromised host, or control over Hump's configuration or environment. The configuration is trusted input: a template can read any environment variable of the process with `env`.
- Deliberate misconfiguration, such as exposing the web port to the internet (see below).

## Deployment hardening

### Web console and telemetry ports

Neither port has authentication. Each is off until you set `--web-port` (`WEB_PORT`) or `--telemetry-port` (`TELEMETRY_PORT`), and it then listens on all interfaces. Hump has no option to pick the bind address.

Anyone who can reach the web port can:

- read the configuration exactly as it was passed at startup (`GET /api/config`);
- read delivered messages with their payloads, which contain row data, and their headers, including an `Authorization` header sent to an HTTP sink (`POST /api/operations`);
- start and stop snapshots (`POST /api/snapshot/start`, `POST /api/snapshot/stop`).

The telemetry port serves `/metrics` and the `/health/*` probes.

Turn the web port on only where you need it. Keep both ports on a private network: a firewall rule, an internal container network, or a reverse proxy that adds authentication.

### PostgreSQL role

Connect as a dedicated role rather than a superuser. Hump needs it to do the following:

- creates the replication slot with `pg_create_logical_replication_slot`, which needs the `REPLICATION` attribute;
- creates the publication or alters it to match the configuration. PostgreSQL requires `CREATE` on the database and ownership of the published tables (or membership in the owning role). Hump alters the publication on every start, so the role must also own it;
- runs `ALTER TABLE … REPLICA IDENTITY FULL` on configured tables with the default `replica: full` that aren't `FULL` yet, which also needs ownership;
- reads snapshots with `COPY (SELECT …)` after `LOCK TABLE … IN ACCESS SHARE MODE`. Both need `SELECT` on the tables.

Hump creates no tables of its own. A TimescaleDB hypertable is published as `TABLES IN SCHEMA "_timescaledb_internal"`, and PostgreSQL allows that form to superusers only.

To encrypt the connection, add `sslmode=verify-full` to the connection URL (plus `sslrootcert` for a private CA).

### Secrets

- Pass the connection URL in `DATABASE_URL` rather than `--database-url`. Other users on the host can see command-line arguments in the process list.
- Hump doesn't substitute environment variables in the YAML file. Credential fields are templates, though, so read them with `env`: `password: '{{ env "KAFKA_PASSWORD" }}'`. This works for Kafka `username` and `password`, NATS `username`, `password`, `token`, `credentials` and `nkeySeed`, RabbitMQ and RabbitMQ Streams `url`, and HTTP `headers`. See [Process environment](docs/template.md#process-environment).
- `GET /api/config` returns the configuration source, so it shows `{{ env "..." }}` as written and an inline password in plain text. The operations log shows rendered header values either way.
- `LOG_VERBOSE=true` logs the parsed configuration at startup, inline credentials included.
- Keep a configuration file that holds secrets out of version control, and make it readable only by the user Hump runs as (`chmod 600`).

### Sink connections

- Kafka: set `tls: true` (TLS 1.2 or later) and `sasl` (`plain`, `scram-sha-256` or `scram-sha-512`) with `username` and `password`.
- HTTP: use `https://` URLs.
