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

## Naming

- Every workflow, job and step has an explicit `name:`. Don't use the tool name or a generic verb
  on its own.
- Workflow: `Area: action`, a short area, a colon and a short lowercase action:
  `Repository: check`, `Workflows: lint`, `Workflows: security report`; not
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
- moon prefixes task output in CI, so a workflow command (`::error ...`) printed by a task does
  not become an annotation. A task never prints workflow commands. It writes them to
  `.ci/annotations/<project>-<check>.txt`, its reports (SARIF, coverage, JUnit) to
  `.ci/<project>/`, and on failure a marker to `.ci/failed/<project>-<check>`. Steps after
  `moon ci` print the annotations, fail on the markers and read the reports.
- A CI-only task is named `ci-<check>` and has `runInCI: 'only'`, `cache: false` and
  `allowFailure: true`, so one failure does not cancel the other checks. Tasks that CI runs
  without `allowFailure` (`hump:tidy`, `hump-tower:openapi`, the builds) fail `moon ci` directly.
- Drift checks are workflow steps after `moon ci`: they diff what the tasks regenerated (tidy
  `go.mod`/`go.sum`, the OpenAPI spec, the embedded UI build, `bun.lock`) and fail with an
  `::error` annotation that names the command to run locally.
- Dependabot directories: `gomod` in `/apps/hump` and `/apps/hump/tools/actionlint`, `bun` in
  `/` (the root `bun.lock` covers every workspace), `docker-compose` in `/dev`,
  `github-actions` in `/`. A new Go module under `packages/` needs its own `gomod` entry.
