# Versioning and compatibility

Hump follows [Semantic Versioning 2.0.0](https://semver.org/). A version is `MAJOR.MINOR.PATCH`:

- **PATCH** (`1.0.1`): bug and security fixes. No new features, no breaking changes.
- **MINOR** (`1.1.0`): new features and deprecations. Existing configurations keep working.
- **MAJOR** (`2.0.0`): breaking changes to the public contract below.

Every release and its changes, including breaking changes and deprecations, are listed in the
[changelog](https://hump.dev/changelog).

## Public contract

These are covered by the compatibility rules. A breaking change to any of them ships only in a
major release:

- configuration keys, their types and defaults;
- CLI flags and environment variables;
- the event envelope and the default payload format;
- the data available to templates;
- Prometheus metric names and labels;
- health probe endpoints;
- exit codes;
- how Hump creates, names, uses and drops its publication and replication slot.

Anything else is internal and can change in any release: the web console and its HTTP API, log
messages and their format, Go packages under `apps/hump`, and the layout of the Docker image apart
from the binary path and entry point.

## Supported environments

| | Supported |
|---|---|
| PostgreSQL | 15 and later, while the PostgreSQL project supports them |
| Sinks | The versions listed in the [documentation](https://hump.dev/docs/getting-started) |

Dropping support for a PostgreSQL major version is a breaking change, so it ships only in a Hump
major release.

## Deprecation

A deprecated part of the public contract keeps working for at least two minor releases and is
removed only in the next major release.

While a deprecated config key, flag or environment variable is in use, Hump logs a warning on start
that names it and its replacement. Every deprecation is announced in the changelog with the release
it was deprecated in and what to use instead.

## Supported releases

Which releases get fixes is described in [SECURITY.md](SECURITY.md#supported-versions). There are
no long-term support releases.

## Docker image tags

`humpcdc/hump` is published with these tags for each release:

- `1.2.3`: that exact release;
- `1.2`: the latest patch of `1.2`;
- `1`: the latest release of major version `1`. It is not published for `0.x` releases.

Pin `1.2.3` or a digest in production. Upgrades within `1` are safe by the rules above.
