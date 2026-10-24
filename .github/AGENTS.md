# .github — rules for workflows

Applies to `.github/workflows/*.yml` and `.github/dependabot.yml`. moon conventions (`ci-*` tasks,
the `.ci/` output contract, `tool` directives), the dependency age rule and adding a Go module are
in the root `AGENTS.md`.

## Workflows

| File | Name | Purpose |
|---|---|---|
| `check.yml` | `Repository: check` | Every lint, build and test task through `moon ci` |
| `workflows-lint.yml` | `Workflows: lint` | zizmor only; actionlint is the `root:ci-actionlint` task |
| `security-scan.yml` | `Repository: security scan` | zizmor, govulncheck and bun audit reports; rescans the latest release image |
| `scorecard.yml` | `Repository: OpenSSF Scorecard` | Scorecard report behind the README badge |
| `scan-image.yml` | `Docker: build and scan` | Reusable: builds the image for both platforms, scans with Trivy |
| `docker-scan.yml` | `Docker: scan` | Calls `scan-image.yml` on every PR and push to `main` |
| `publish.yml` | `Docker: publish` | On a `v*` release: calls `scan-image.yml`, `publish-image.yml`, then `scan-published-image.yml` on the pushed digest |
| `publish-image.yml` | `Docker: publish image` | Reusable: pushes, signs and attests `docker.io/humpcdc/hump` |
| `scan-published-image.yml` | `Docker: scan published image` | Reusable: scans the pushed image by digest (or the latest release image) with Trivy |

Don't break these:

- `scorecard.yml` follows the Scorecard [restrictions](https://github.com/ossf/scorecard-action#workflow-restrictions),
  or the API rejects its results: no top-level `env`/`defaults`, no workflow-level write
  permissions, `id-token: write` only in the Scorecard job. That job has no `env`, `defaults`,
  `container` or `services`, runs on an Ubuntu hosted runner and uses only `actions/checkout`,
  `actions/upload-artifact`, `github/codeql-action/upload-sarif`, `ossf/scorecard-action` and
  `step-security/harden-runner`. No `run:` steps.
- The path of `publish-image.yml` is the Sigstore signer identity (SLSA Build L3),
  `hump-cdc/hump/.github/workflows/publish-image.yml@refs/tags/v*`. Rename or move it only
  together with the verification docs.
- Caller permissions cap the called workflow, so the `Publish` job grants `contents: read`,
  `id-token: write` and `attestations: write`. Secrets are scoped to environment `production`;
  the caller passes none. Docker workflows use no moon, no repository scripts and no cache.
- Reusable workflows are called by local path with `# zizmor: ignore[self-repository]`
  (actionlint v1.7.12 lacks `$/...`). They set their own `permissions: {}`, `defaults` and job
  `timeout-minutes`, and no `concurrency` (the caller's group applies).
- Fork and Dependabot PRs get a read-only token, so the Trivy SARIF upload is skipped there.
  Trivy exceptions go to `.trivyignore` in the repo root (Trivy picks it up on its own; create it
  when needed), each with a reason and a review date.
- Dependabot doesn't bump these inputs; raise them by hand to releases at least 7 days old:
  `aquasecurity/trivy-action` `version`, `sigstore/cosign-installer` `cosign-release`,
  `codecov/codecov-action` `version`. zizmor-action has no `version` input set: its default
  `latest` is the zizmor version the action locks with hashes, so it follows the action SHA.
- Drift checks are steps after `moon ci`: they diff regenerated files and fail with an `::error`
  naming the local command to run. Tasks CI runs without `allowFailure` (`hump:tidy`,
  `hump-tower:openapi`, the builds) fail `moon ci` directly.
- The `main` ruleset's code scanning rule waits for tool `zizmor` on every PR commit, so
  `workflows-lint.yml` (PRs) and `security-scan.yml` (`main`) both upload zizmor SARIF with
  category `zizmor`. In SARIF mode zizmor exits 0 on findings; the ruleset blocks on error-level
  or high alerts. A fork PR can't upload, so there `workflows-lint.yml` runs zizmor without SARIF
  and fails on findings; its required `Security` check blocks the merge.
- A new `gomod` entry in `dependabot.yml` copies the cooldown and group of the existing ones.

## Pinning actions

- Every `uses:` is a full 40-character commit SHA plus a comment with the exact release tag.
  Never use a tag or branch as the ref:
  `uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1`.
- Resolve the SHA with `gh api repos/<owner>/<repo>/git/ref/tags/<tag>`. For an annotated tag
  (`"type": "tag"`), dereference it with `gh api repos/<owner>/<repo>/git/tags/<sha>`. Change the
  tag comment together with the SHA. Dependabot bumps both; don't bypass it with a mutable ref.
- Lint workflows locally with `moon run root:actionlint`.

## Runner hardening

- `step-security/harden-runner` is the first step of every job, reusable workflows included.
  Composite actions under `.github/actions` are not jobs and don't get it.
- A new job starts with `egress-policy: audit`. Once audit runs show its endpoints, switch to
  `egress-policy: block` with `allowed-endpoints` (`host:443`) and rename the step to `Restrict
  outbound network traffic of the runner to the allowed endpoints`. Current state:
  `grep -n egress-policy .github/workflows/*.yml`.
- List only what the audit runs show. The agent allows GitHub Actions and StepSecurity endpoints
  itself. Telemetry is disabled, not allowed: `telemetry` in `.prototools` and
  `.moon/workspace.yml`, `disable_telem: true` on every `codecov/codecov-action` step.
- A job running proto or moon under `block` sets job-level `env: PROTO_OFFLINE: "0"`, or proto's
  dropped connectivity probes break tool installs. Don't allow the probe hosts (`google.com`,
  `clients3.google.com`, `detectportal.firefox.com`).
- On a blocked connection, open the insights link the harden-runner step prints in the log and
  add the endpoint to that job's list.

## Naming

- Every workflow, job and step has an explicit `name:`, never a bare tool name or generic verb.
- Workflow: `Area: action` (`Repository: check`, `Workflows: lint`). Job: a short action without
  the area, since GitHub shows `Workflow / Job` (`Check`, `Security report`; not `zizmor`). Step:
  the full action and subject (`Lint workflow syntax, expressions and shell scripts`, not `Run
  actionlint`).
- Job names are the required checks of the `main` ruleset; renaming a job means updating it. A
  reusable call shows as `caller job / called job`. Required: `Check`, `Docker image / Build and
  scan` and `Security`; the ruleset has no bypass actors.
- The ruleset matches checks by job name alone, so two jobs share a name only if they should
  count as one check.

## Failure messages

- Every failure says what is wrong and how to fix it, so the developer can act on the message
  alone. A bare non-zero exit or a raw tool dump is not enough.
- In a custom `run:` check, emit an error annotation before `exit 1`: point it at the file, title
  it with the problem, put the fix in the message. Print details (diff, values) to the log too.

  ```bash
  echo "::error file=go.mod,title=go.mod and go.sum are not tidy::Run 'go mod tidy' locally and commit the updated go.mod and go.sum."
  ```
