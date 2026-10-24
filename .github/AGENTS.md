# .github — rules for workflows

Applies to `.github/workflows/*.yml` and `.github/dependabot.yml`.

## Pinning action versions

- Every `uses:` points to a full 40-character commit SHA, followed by a comment with the exact
  release tag. Tags and branches are mutable and are never used as refs.

  ```yaml
  uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
  ```

- Resolve the SHA from the release tag: `gh api repos/<owner>/<repo>/git/ref/tags/<tag>`. If the
  returned object is an annotated tag (`"type": "tag"`), dereference it to the commit with
  `gh api repos/<owner>/<repo>/git/tags/<sha>`. Change the comment whenever you change the SHA.
- Dependabot (the `github-actions` ecosystem in `.github/dependabot.yml`) bumps the SHA and the
  comment together. Don't bypass it with a mutable ref.
- zizmor is pinned through the `version` input of `zizmorcore/zizmor-action`. The action accepts
  only zizmor versions it ships a lockfile for, so raise `version` only to a version the pinned
  action release supports. Dependabot does not bump this input.
- actionlint is pinned as a Go tool in its own module, `apps/hump/tools/actionlint/go.mod`. It
  can't live in `apps/hump/go.mod` because its dependencies conflict with golangci-lint's, so the
  module stays out of `go.work`. Run it with `moon run root:actionlint`, which calls
  `GOWORK=off go -C apps/hump/tools/actionlint tool actionlint` (`-modfile` is rejected in
  workspace mode).
- A `tool` directive in a `go.mod` must be run as `go tool <name>` in a `moon.yml`, an
  `apps/*/scripts` or `packages/*/scripts` script or `.github`. `check.yml` fails on an unused one in `apps/hump/go.mod`.

## Runner hardening

- The first step of every job, including the jobs of reusable workflows, is
  `step-security/harden-runner`. It starts in `egress-policy: audit`, which only records the
  outbound connections of the job. Once the audit runs (StepSecurity insights) show which
  endpoints a job needs, the job switches to `egress-policy: block` with an `allowed-endpoints`
  list (`host:443`) and the step is renamed to `Restrict outbound network traffic of the runner
  to the allowed endpoints`. A new job gets the audit step. Composite actions under
  `.github/actions` are not jobs and don't get it.
- In `block`: `check.yml` (job `Check`) and `workflows-lint.yml` (job `Security`). In `audit`,
  until their audit runs are collected: `security-scan.yml`, `scorecard.yml` and the reusable
  `scan-image.yml` and `publish-image.yml` (`docker-scan.yml` and `publish.yml` only call them
  and have no steps of their own).
- The agent allows GitHub's Actions service endpoints (`*.actions.githubusercontent.com`,
  `codeload.github.com`, the results storage, the cache host) and the StepSecurity API on its
  own; don't list them. List only what the audit runs show. Telemetry is turned off rather than
  allowed: `telemetry` in `.prototools` and `.moon/workspace.yml`, `disable_telem: true` on
  every `codecov/codecov-action` step. When a step fails on a blocked
  connection, open the insights link that the harden-runner step prints in the log and add the
  endpoint to that job's list.

## Naming

- Every workflow, job and step has an explicit `name:`. Don't use the tool name or a generic verb
  on its own.
- Workflow: `Area: action`, a short area, a colon and a short lowercase action:
  `Repository: check`, `Workflows: lint`, `Repository: security scan`; not
  `GitHub Actions workflows lint`.
- Job: a short action without the area, because GitHub shows checks as `Workflow / Job`
  (`Repository: check / Check`): `Check`, `Security`, `Security report`; not `zizmor`.
- Step: the full action and its subject: `Lint workflow syntax, expressions and shell scripts`,
  not `Run actionlint`.
- A job that calls a reusable workflow shows as `caller job / called job`, so the check of
  `docker-scan.yml` is `Docker image / Build and scan`; renaming either job renames the check.
- Job names are the status-check names that the branch ruleset on `main` lists as required
  checks. Renaming a job requires updating the ruleset.
- The ruleset matches a required check by job name alone, so two jobs must not share a name
  unless they are meant to count as one check.

## Failure messages

- Every failure a workflow raises must tell the developer what is wrong and how to fix it, so
  they can act on the message alone, without reading the workflow or guessing. A bare non-zero
  exit or a raw tool dump is not enough.
- In a custom `run:` check, emit a GitHub error annotation before `exit 1`: point it at the
  offending file, give it a title that names the problem, and put the fix (the command to run
  or the change to make) in the message. Print the details (diff, offending values) to the log
  as well.

  ```bash
  echo "::error file=go.mod,title=go.mod and go.sum are not tidy::Run 'go mod tidy' locally and commit the updated go.mod and go.sum."
  ```

## Workflows and moon

- `check.yml` (`Repository: check`, job `Check`) runs every lint, build and test task through
  `moon ci`, which picks the projects affected since the base commit. `workflows-lint.yml`
  runs only zizmor; actionlint is the `root:ci-actionlint` task.
- `security-scan.yml` (`Repository: security scan`) runs on push to `main`, weekly (Monday
  06:00 UTC) and on `workflow_dispatch`, and uploads SARIF to code scanning. Job `Security
  report` runs zizmor (category `zizmor`; the `main` ruleset's code scanning rule matches the
  tool `zizmor`). Job `Go vulnerability report` runs `hump:ci-vulncheck` on `apps/hump`, so
  govulncheck also runs weekly, not only when a change affects the module; `ci.sh` writes the
  report with repository-relative paths, the job uploads it under category `govulncheck` and
  then fails on the `.ci/failed/hump-vulncheck` marker or a missing report. Job `npm
  vulnerability report` runs `hump-tower:ci-audit` (bun audit of the root `bun.lock`, direct and
  transitive, plus a check that every package comes from registry.npmjs.org) weekly, not only
  when a change affects the lockfile. bun audit has no SARIF, so nothing is uploaded: the job
  prints `.ci/annotations` and fails on the `.ci/failed/hump-tower-audit` marker or a missing
  `.ci/hump-tower/audit.json`.
- `scorecard.yml` (`Repository: OpenSSF Scorecard`, job `Scorecard report`) runs
  `ossf/scorecard-action` on push to `main`, weekly (Thursday 06:00 UTC) and on
  `workflow_dispatch`. It uploads SARIF to code scanning (category `scorecard`) and, with
  `publish_results: true`, publishes the results to the Scorecard API behind the README badge.
  The API rejects results from a workflow that breaks its
  [restrictions](https://github.com/ossf/scorecard-action#workflow-restrictions): no top-level
  `env` or `defaults`, no workflow-level write permissions, `id-token: write` only in the
  Scorecard job, and that job has no `env`, `defaults`, `container` or `services`, runs on an
  Ubuntu hosted runner and uses only `actions/checkout`, `actions/upload-artifact`,
  `github/codeql-action/upload-sarif`, `ossf/scorecard-action` and
  `step-security/harden-runner`. Don't add `run:` steps or other actions to it.
- `scan-image.yml` (`Docker: build and scan`, reusable, job `Build and scan`) builds
  `apps/hump/Dockerfile` for linux/amd64 and linux/arm64 without pushing (no environment, no
  secrets, no cache), exports each platform to a docker-archive tarball and scans it with Trivy
  (fixable CRITICAL/HIGH, vuln scanner only). It uploads SARIF under categories
  `trivy-linux-amd64` and `trivy-linux-arm64` and fails on findings; exceptions go to
  `.trivyignore` with a reason and a review date. On a PR from a fork or from Dependabot the
  token is read-only, so the upload is skipped there and the findings are only in the log. A
  PR and a release call the same workflow, so both are scanned the same way.
- `docker-scan.yml` (`Docker: scan`) calls `scan-image.yml` on every PR and push to `main`, with
  no paths filter. Its check `Docker image / Build and scan` (caller job / called job) is a
  required check of the `main` ruleset.
- `publish.yml` (`Docker: publish`) runs only on a published release with a `v*` tag. Job `Scan`
  calls `scan-image.yml`. Job `Publish` calls `publish-image.yml` (`Docker: publish image`,
  reusable, job `Build, push, sign and attest`), which runs in environment `production` (secret
  `DOCKERHUB_TOKEN`, variable `DOCKERHUB_USERNAME`, both environment-scoped, so the caller passes
  no secrets) and pushes `docker.io/humpcdc/hump` for linux/amd64 and linux/arm64 with an SBOM
  and `mode=max` provenance, then adds a keyless cosign signature and GitHub build provenance
  and SBOM attestations (`actions/attest`). Signing in a reusable workflow gives SLSA Build L3:
  the Sigstore identity of the signature and the attestations is
  `hump-cdc/hump/.github/workflows/publish-image.yml@refs/tags/v*`, so don't rename or move the
  file without updating the verification docs. A caller's permissions are the ceiling of the
  called workflow, so the `Publish` job grants the same `contents: read`, `id-token: write` and
  `attestations: write`. No moon, no repository scripts and no cache in these workflows.
  Dependabot does not bump the `version` input of `aquasecurity/trivy-action` or the
  `cosign-release` input of `sigstore/cosign-installer`; raise them by hand to releases at least
  7 days old.
- Reusable workflows are called by the local path (`uses: ./.github/workflows/...`) with
  `# zizmor: ignore[self-repository]`, because actionlint v1.7.12 does not support the `$/...`
  form yet. They set their own `permissions: {}`, `defaults` and job `timeout-minutes` and no
  `concurrency` (the caller's group applies).
- The bun and Go versions in the `FROM` lines of `apps/hump/Dockerfile` must match
  `.moon/toolchains.yml`. Dependabot updates only the Dockerfile, so align them by hand.
- moon prefixes task output in CI, so a workflow command (`::error ...`) printed by a task does
  not become an annotation. A task never prints workflow commands. It writes them to
  `.ci/annotations/<project>-<check>.txt`, its reports (SARIF, coverage, JUnit) to
  `.ci/<project>/`, and on failure a marker to `.ci/failed/<project>-<check>`. Steps after
  `moon ci` print the annotations, fail on the markers and read the reports.
- A CI-only task is named `ci-<check>` and has `runInCI: 'only'`, `cache: false` and
  `allowFailure: true`, so one failure does not cancel the other checks. Tasks that CI runs
  without `allowFailure` (`hump:tidy`, `hump-tower:openapi`, the builds) fail `moon ci` directly.
  `hump:ci-dep-age` is a CI-only task: it fails on a Go module published less than 7 days ago,
  matching the Dependabot cooldown (details in the root `AGENTS.md`, "Dependencies").
- Drift checks are workflow steps after `moon ci`: they diff what the tasks regenerated (tidy
  `go.mod`/`go.sum`, the OpenAPI spec, the embedded UI build, `bun.lock`) and fail with an
  `::error` annotation that names the command to run locally.
- Dependabot directories: `gomod` in `/apps/hump`, `/apps/hump/tools/actionlint` and
  `/apps/hump/tools/govulncheck`, `bun` in `/` (the root `bun.lock` covers every workspace),
  `docker-compose` in `/dev`, `docker` in `/apps/hump` (the `FROM` digests of the Dockerfile),
  `github-actions` in `/`. A new Go module under `packages/` or
  `apps/hump/tools/` needs its own `gomod` entry and must be added to the dep-age check in
  `apps/hump/scripts/ci.sh`.
- Dependabot cooldown: `default-days: 7` everywhere; `gomod` and `bun` also wait 30 days for a
  new major version (`semver-major-days: 30`). `github-actions`, `docker-compose` and `docker`
  don't support the SemVer cooldown options, so they keep 7 days for every update.
- Dependabot groups: `gomod` and `bun` entries group minor and patch updates into one PR per
  directory (`minor-patch`); each major update gets its own PR. `github-actions` groups every
  action update into one PR. A new `gomod` entry copies the cooldown and the group.
