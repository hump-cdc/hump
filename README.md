<div align="center">

<img src="apps/hump-tower/public/logo_lg.svg" width="120" alt="Hump logo">

<h1>Hump</h1>

<p>Your database, as an event stream.</p>

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/hump-cdc/hump)](https://github.com/hump-cdc/hump/releases)
![PostgreSQL 15+](https://img.shields.io/badge/PostgreSQL-15%2B-336791)

[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/hump-cdc/hump/badge)](https://scorecard.dev/viewer/?uri=github.com/hump-cdc/hump)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/15877/badge)](https://www.bestpractices.dev/projects/15877)
[![SLSA 3](https://slsa.dev/images/gh-badge-level3.svg)](#verifying-the-image)

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

## Docker image

The image is `docker.io/humpcdc/hump`, built for linux/amd64 and linux/arm64. Each release gets three tags: the full version (`1.2.3`), `major.minor` (`1.2`) and `major` (`1`, skipped for 0.x). Pre-releases do not move `latest`.

Pin the image by digest, e.g. `docker.io/humpcdc/hump:1.2.3@sha256:<digest>`. To get the digest:

```sh
docker buildx imagetools inspect docker.io/humpcdc/hump:<version>
```

### Verifying the image

Images are published from the `v*` release tags of `hump-cdc/hump` by the reusable workflow `.github/workflows/call-docker-publish.yml`. It signs the multi-arch index digest with keyless cosign (Sigstore, GitHub OIDC) and pushes two attestations to the registry: GitHub build provenance (SLSA Build L3) and an SPDX SBOM. BuildKit attaches its own SBOM and provenance as well. Trivy rescans the published image right after the push, and the latest release image again every week. The release workflow also runs the signature and attestation checks below on every published digest.

Check the signature:

```sh
cosign verify docker.io/humpcdc/hump@sha256:<digest> \
  --certificate-identity-regexp '^https://github\.com/hump-cdc/hump/\.github/workflows/call-docker-publish\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

Check the build provenance. `--signer-workflow` accepts only attestations signed by the release workflow, so one made by another workflow in the repository fails the check. Add `--source-ref refs/tags/v<version>` to also pin the release tag:

```sh
gh attestation verify oci://docker.io/humpcdc/hump@sha256:<digest> \
  --repo hump-cdc/hump \
  --signer-workflow hump-cdc/hump/.github/workflows/call-docker-publish.yml \
  --deny-self-hosted-runners
```

To check the SBOM attestation, run the same command with `--predicate-type https://spdx.dev/Document/v2.3`.

## Documentation

The full documentation is at [hump.dev/docs](https://hump.dev/docs/getting-started). It covers configuration, sinks, templating, delivery guarantees, snapshots and observability.

Versioning, compatibility and deprecation rules are in [VERSIONING.md](VERSIONING.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to propose changes, set up the dev environment and open a pull request.

## License

Hump is released under the [MIT License](LICENSE).
