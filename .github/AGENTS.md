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

- The first step of every job is `step-security/harden-runner` with `egress-policy: audit`.
  Audit mode only records the outbound connections of the job. Once the audit runs show which
  endpoints each job needs, the policy switches to `block` with an `allowed-endpoints` list per
  job. A new job gets the same first step. Composite actions under `.github/actions` are not
  jobs and don't get it.

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
  `docker-compose` in `/dev`, `github-actions` in `/`. A new Go module under `packages/` or
  `apps/hump/tools/` needs its own `gomod` entry and must be added to the dep-age check in
  `apps/hump/scripts/ci.sh`.
- Dependabot cooldown: `default-days: 7` everywhere; `gomod` and `bun` also wait 30 days for a
  new major version (`semver-major-days: 30`). `github-actions` and `docker-compose` don't
  support the SemVer cooldown options, so they keep 7 days for every update.
- Dependabot groups: `gomod` and `bun` entries group minor and patch updates into one PR per
  directory (`minor-patch`); each major update gets its own PR. `github-actions` groups every
  action update into one PR. A new `gomod` entry copies the cooldown and the group.
