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
| `apps/hump` | Go module `hump`: `cmd/`, `internal/`, `pkg/`, `config/`, `tools/actionlint` and `tools/govulncheck` (separate Go tool modules, outside `go.work`), `Dockerfile` (image `humpcdc/hump`, build context: repo root) |
| `apps/hump-tower` | Web UI, package `hump-tower` |
| `packages/` | Shared packages; empty for now |
| `dev/` | Local dev environment: `docker-compose.yml`, `hump.yaml` |
| `docs/` | Design notes |
| `.moon/` | `workspace.yml` (project globs), `toolchains.yml` (proto, Go, Bun versions) |
| `moon.yml` | The `root` project: repo-wide tasks |
| `go.work` | Go workspace: `use ./apps/hump` |
| `package.json`, `bunfig.toml`, `bun.lock` | Bun workspace over `apps/*` and `packages/*`; one lockfile for all |
| `.dockerignore` | Build context filter for `apps/hump/Dockerfile` |
| `.github/` | Workflows, Dependabot, composite actions |

## moon

Projects: `root` (`moon.yml`), `hump` (`apps/hump/moon.yml`), `hump-tower`
(`apps/hump-tower/moon.yml`). Every project matching `apps/*/moon.yml` or
`packages/*/moon.yml` is picked up.

- Run a task from any directory: `moon run <project>:<task>`, e.g. `moon run hump:test`.
- CI runs the full set of checks in explicit Actions steps on every run, without affected-project
  selection. It calls the same moon commands used locally and passes report arguments as needed.
- Task names are kebab-case (`test-file-cover`) and every task has a `description`.
- Keep real checks inline in `moon.yml`: use `command:` for simple invocations with passthrough
  arguments, and `script:` for multi-step actions with explicit argument handling. Put CI
  orchestration and reporting inline in Actions, not in shell wrappers.
- A task belongs to the project whose files it works on. Repo-wide tasks (workflow lint, the
  dev environment, the aggregate `check`) go to the root `moon.yml`. Exception:
  `hump-tower:openapi` runs Go code from `apps/hump` but lives in `hump-tower`, because a
  dependency back into `hump` would make a cycle (`apps/hump-tower/moon.yml`).
- Developer tasks (`run`, `dev`, `lint-fix`, `test-cover` etc.) have `runInCI: false`; the
  ones that are not cached use `preset: 'utility'` or `'server'`.
- Each moon command performs one action and returns its real exit status. Do not add CI-only
  wrappers, report-only variants, GitHub annotations or failure markers to moon tasks.
- Local output is readable by default. Actions supplies report formats and paths; `.ci/<project>/`
  (gitignored) may hold SARIF, coverage and JUnit reports. Actions handles annotations, summaries
  and uploads. Independent checks continue after a failure, but the job still fails; dependent
  steps require successful preparation. Details in `.github/AGENTS.md`.
- Security and policy checks are ordinary local commands: `moon run hump:vulncheck`,
  `moon run hump:dep-age`, `moon run hump-tower:audit` and `moon run hump-tower:registry-check`.
  Do not cache advisory or dependency-age checks. CI must execute checks and produce fresh reports.

### Adding a package

1. Create `packages/<name>/` with a `moon.yml` (`language`, `layer`, `project.title`,
   `project.description`, tasks).
2. TypeScript: add a `package.json`, then run `bun install` in the root to update `bun.lock`.
3. Go: add a `go.mod` and its directory to `go.work` (`go work use ./packages/<name>`). Add a
   `gomod` entry for it to `.github/dependabot.yml` and add the module to the dep-age check
   (`hump:dep-age`) in `apps/hump/moon.yml`. A Go tool module kept outside `go.work`
   (like `apps/hump/tools/actionlint` and `apps/hump/tools/govulncheck`) gets the same
   Dependabot entry and dep-age check, plus a `GOWORK=off go -C <dir> mod tidy` in the
   `hump:tidy` task and its `go.mod` / `go.sum` in the tidy diff of `.github/workflows/ci-check.yml`.

## Dependencies

Do not add or upgrade a dependency, npm, Go or toolchain, to a version published less than 7
days ago. This matches the 7-day Dependabot cooldown in `.github/dependabot.yml`. Pin exact
versions everywhere.

- npm (bun): `install.minimumReleaseAge` in the root `bunfig.toml` enforces the age. `bun add` /
  `bun update` silently skip younger versions and fail only when no older one matches (e.g. an
  exact too-new version). Entries already in `bun.lock` are not re-checked, so do not hand-edit
  `bun.lock`.
  No `^`, `~`, `*` or `latest` in any workspace `package.json` (root, `apps/*`, `packages/*`).
  `install.exact` makes `bun add` / `bun update` save exact versions; it does not check what you
  type by hand.
  No dependency runs install scripts: `"trustedDependencies": []` in the root `package.json`
  replaces Bun's default allow list (it blocks e.g. esbuild's `postinstall`). To allow one, review
  its script and list that package there explicitly.
- Go: `hump:dep-age` fails when any module in the full build list (`go list -m all` with
  `GOWORK=off`, transitive included) of `apps/hump`, `apps/hump/tools/actionlint` or
  `apps/hump/tools/govulncheck` is too young. The last one pins govulncheck (`golang.org/x/vuln`),
  which `hump:vulncheck` builds from it. Run the age check locally with `moon run hump:dep-age`.
  Before `go get` of a version, check its `Time` with `go list -m -json <module>@<version>`.
  A `tool` directive in a `go.mod` must be run as `go tool <name>` in a `moon.yml` or
  inline in a workflow or composite action under `.github`; CI fails on an unused one in
  `apps/hump/go.mod`. Remove it with `go get -tool <path>@none` and `go mod tidy`.
- Toolchain: moon and Go in `.prototools`; proto, Go and Bun in `.moon/toolchains.yml`. Go is
  pinned in `.prototools` because `proto install` otherwise reads the `go` directive of `go.mod` /
  `go.work` as a `^` range and installs a newer Go. Dependabot does not update these files. It does
  bump the base images in `apps/hump/Dockerfile`, whose bun tag must match `.moon/toolchains.yml`.
  The Go version must match in `.moon/toolchains.yml`, `.prototools`, the `golang` tag in
  `apps/hump/Dockerfile` and the `go` directive of `apps/hump/go.mod` and `go.work`; change them
  together.
  `.prototools` sets `auto-install`; proto applies it only after `proto trust .prototools`.

## Safe logging

- System logs at every level, including Debug, must not contain secrets, database row contents,
  payloads or sensitive headers. Log only allowlisted diagnostic metadata. This rule applies to
  system logs, not intentional web operations capture.
- Format diagnostic URLs with `apps/hump/pkg/observability/diagnostic`. Check wrapped and nested
  errors and shutdown-hook names for leaks too.
- Cover new system logging paths with secret-sentinel regression tests.

## Before finishing a task

Every time you finish a task, run the checks and fix everything they report before you call
the task done. `moon run root:check` runs them all: workflow lint, Go lint and tests, UI lint,
tests, type check and knip. Or run the tasks of the project you changed:

- Go: `moon run hump:lint` (golangci-lint; the layer rules it enforces are in
  `apps/hump/AGENTS.md`). `moon run hump:lint-fix` applies the automatic fixes.
- Web UI: `moon run hump-tower:check` (Biome: lint, format, import order);
  `bun run check --write` in `apps/hump-tower` applies the safe fixes. Then
  `moon run hump-tower:knip` (unused files, exports, types and dependencies); it must exit 0.
