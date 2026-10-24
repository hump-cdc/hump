# hump

Change data capture for PostgreSQL: hump reads logical replication (WAL) changes — insert,
update, delete, truncate — for the configured tables and delivers them to sinks: Kafka, HTTP,
NATS (core / JetStream), RabbitMQ, RabbitMQ Streams. Payloads are templated; delivery has
retry and failure policies. It can also snapshot existing table rows, exposes Prometheus
metrics, and ships a web UI. `README.md` has a quick start; the full configuration reference is
at https://hump.dev/docs/getting-started, templating is in `docs/template.md`.

## Stack

- Go application `apps/hump` (module `hump`), entry point `apps/hump/cmd/hump`.
- Web UI `apps/hump-tower` on React (TypeScript, rsbuild, bun). Its build output goes to
  `apps/hump/internal/controller/hump_tower` and is embedded into the binary.
- A moon (v2) monorepo: moon runs every task, proto pins the toolchain.

Project rules: `apps/hump/AGENTS.md` (Go app), `apps/hump-tower/AGENTS.md` (web UI),
`.github/AGENTS.md` (workflows).

## Layout

| Path | Contents |
|---|---|
| `apps/hump` | Go module `hump`: `cmd/`, `internal/`, `pkg/`, `config/`, `scripts/`, `tools/actionlint` (separate Go module, outside `go.work`) |
| `apps/hump-tower` | Web UI, package `hump-tower` |
| `packages/` | Shared packages; empty for now |
| `dev/` | Local dev environment: `docker-compose.yml`, `hump.yaml` |
| `docs/` | Design notes |
| `.moon/` | `workspace.yml` (project globs), `toolchains.yml` (proto, Go, Bun versions) |
| `moon.yml` | The `root` project: repo-wide tasks |
| `go.work` | Go workspace: `use ./apps/hump` |
| `package.json`, `bunfig.toml`, `bun.lock` | Bun workspace over `apps/*` and `packages/*`; one lockfile for all |
| `.github/` | Workflows, Dependabot, composite actions |

## moon

Projects: `root` (`moon.yml`), `hump` (`apps/hump/moon.yml`), `hump-tower`
(`apps/hump-tower/moon.yml`). Every project matching `apps/*/moon.yml` or
`packages/*/moon.yml` is picked up.

- Run a task from any directory: `moon run <project>:<task>`, e.g. `moon run hump:test`.
- CI runs `moon ci`: every task with `runInCI` in the projects affected since the base commit.
  Run a CI-only task locally with `CI=true moon run <project>:<task>`.
- Task names are kebab-case (`test-file-cover`) and every task has a `description`.
- A task belongs to the project whose files it works on. Repo-wide tasks (workflow lint, the
  dev environment, the aggregate `check`) go to the root `moon.yml`.
- Developer tasks (`run`, `dev`, `lint-fix`, `test-cover` etc.) have `runInCI: false`; the
  ones that are not cached use `preset: 'utility'` or `'server'`.
- `ci-*` tasks exist only for CI: `runInCI: 'only'`, `cache: false`, `allowFailure: true`.
  A failed one does not cancel the other checks; it leaves a marker that the workflow fails on.
- moon prefixes task output in CI, so workflow commands (`::error ...`) printed by a task do not
  become annotations. Tasks write to `.ci/` (gitignored) instead:
  - `.ci/<project>/` — reports: SARIF, coverage, JUnit;
  - `.ci/annotations/<project>-<check>.txt` — annotations and fix hints;
  - `.ci/failed/<project>-<check>` — failure markers.

  `.github/workflows/check.yml` reads them after `moon ci`, and `.github/workflows/security-scan.yml`
  after its weekly `hump:ci-vulncheck` run. Details in `.github/AGENTS.md`.

### Adding a package

1. Create `packages/<name>/` with a `moon.yml` (`language`, `layer`, `project.title`,
   `project.description`, tasks).
2. TypeScript: add a `package.json`, then run `bun install` in the root to update `bun.lock`.
3. Go: add a `go.mod` and its directory to `go.work` (`go work use ./packages/<name>`). Add a
   `gomod` entry for it to `.github/dependabot.yml` and add the module to the dep-age check
   (`hump:ci-dep-age`) in `apps/hump/scripts/ci.sh`.

## Dependencies

Do not add or upgrade a dependency to a version published less than 7 days ago. This matches
the 7-day Dependabot cooldown in `.github/dependabot.yml`.

- npm (bun): enforced by `install.minimumReleaseAge` in the root `bunfig.toml`. When resolving
  (`bun add`, `bun update`), bun silently skips versions younger than 7 days and fails only when
  no older version matches (e.g. an exact too-new version). Entries already in `bun.lock` are
  not re-checked, so do not hand-edit `bun.lock`.
  Pin every dependency in every workspace `package.json` (root, `apps/*`, `packages/*`) to an
  exact version: no `^`, `~`, `*` or `latest`. `install.exact` in the root `bunfig.toml` makes
  `bun add` / `bun update` save exact versions, but it does not check what you type into
  `package.json` by hand, so write those exact too.
- Go: enforced in CI by `hump:ci-dep-age`. It fails when any module in the full build list
  (`go list -m all`, direct and transitive, with `GOWORK=off`) of `apps/hump/go.mod` or
  `apps/hump/tools/actionlint/go.mod` was published less than 7 days ago. It also checks
  `golang.org/x/vuln` at `GOVULNCHECK_VERSION`, the govulncheck version pinned in
  `apps/hump/scripts/ci.sh`, so raising that version falls under the rule too. Run it locally
  with `CI=true moon run hump:ci-dep-age`.
  Pin concrete versions in every `go.mod` (`apps/hump`, `apps/hump/tools/actionlint`, future
  `packages/*`). Before `go get` of a version, check its publish time with
  `go list -m -json <module>@<version>` (the `Time` field) and pick one at least 7 days old,
  so CI does not fail after the fact.
  A `tool` directive in a `go.mod` must be run as `go tool <name>` in a `moon.yml`, an
  `apps/*/scripts` (or `packages/*/scripts`) script or `.github`; CI fails on an unused one in
  `apps/hump/go.mod`. Remove it with `go get -tool <path>@none` and `go mod tidy`.
- Toolchain: moon in `.prototools`, proto, Go and Bun in `.moon/toolchains.yml`, all exact
  versions. The same 7-day rule applies. Dependabot does not update these files.
  `.prototools` sets `auto-install`; proto applies it only after `proto trust .prototools`.

## Before finishing a task

Every time you finish a task, run the checks and fix everything they report before you call
the task done. `moon run root:check` runs them all: workflow lint, Go lint and tests, UI lint,
tests, type check and knip. Or run the tasks of the project you changed:

- Go: `moon run hump:lint` (golangci-lint; the layer rules it enforces are in
  `apps/hump/AGENTS.md`). `moon run hump:lint-fix` applies the automatic fixes.
- Web UI: `moon run hump-tower:check` (Biome: lint, format, import order);
  `bun run check --write` in `apps/hump-tower` applies the safe fixes. Then
  `moon run hump-tower:knip` (unused files, exports, types and dependencies); it must exit 0.
