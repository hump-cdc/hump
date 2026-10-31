# .github — rules for workflows

Applies to `.github/workflows/*.yml`, `.github/actions/*/action.yml` and `.github/dependabot.yml`.
moon conventions (local commands, report arguments, `tool` directives), the dependency age rule and adding a Go module are
in the root `AGENTS.md`.

## Workflows

| File | Name | Purpose |
|---|---|---|
| `ci-check.yml` | `Repository: check` | One job per moon project (`hump`, `hump-tower`, `root`); explicit steps run the full set of checks every time |
| `pr-workflows-lint.yml` | `Workflows: lint` | zizmor only; actionlint is the `root:actionlint` task |
| `main-security-scan.yml` | `Repository: security scan` | zizmor, govulncheck and bun audit reports; rescans the latest release image |
| `main-scorecard.yml` | `Repository: OpenSSF Scorecard` | Scorecard report behind the README badge |
| `call-docker-scan.yml` | `Docker: build and scan` | Reusable: builds the image for both platforms, scans with Trivy |
| `ci-docker-scan.yml` | `Docker: scan` | Calls `call-docker-scan.yml` on every PR and push to `main` |
| `release-docker-publish.yml` | `Docker: publish` | On a `v*` release: calls `call-docker-scan.yml`, `call-docker-publish.yml`, then `call-docker-scan-published.yml` and `call-docker-verify-published.yml` on the pushed digest |
| `call-docker-publish.yml` | `Docker: publish image` | Reusable: pushes, signs and attests `docker.io/humpcdc/hump` |
| `call-docker-scan-published.yml` | `Docker: scan published image` | Reusable: scans the pushed image by digest (or the latest release image) with Trivy |
| `call-docker-verify-published.yml` | `Docker: verify published image` | Reusable: runs the README checks on the pushed digest: cosign signature, build provenance and SBOM attestations |

Don't break these:

- `main-scorecard.yml` follows the Scorecard [restrictions](https://github.com/ossf/scorecard-action#workflow-restrictions),
  or the API rejects its results: no top-level `env`/`defaults`, no workflow-level write
  permissions, `id-token: write` only in the Scorecard job. That job has no `env`, `defaults`,
  `container` or `services`, runs on an Ubuntu hosted runner and uses only `actions/checkout`,
  `actions/upload-artifact`, `github/codeql-action/upload-sarif`, `ossf/scorecard-action` and
  `step-security/harden-runner`. No `run:` steps.
- The path of `call-docker-publish.yml` is the Sigstore signer identity (SLSA Build L3),
  `hump-cdc/hump/.github/workflows/call-docker-publish.yml@refs/tags/v*`. Rename or move it only
  together with the verification docs.
- Caller permissions cap the called workflow, so the `Publish` job grants `contents: read`,
  `id-token: write` and `attestations: write`. Secrets are scoped to environment `production`,
  but the caller must pass `secrets: inherit`: without it the called workflow's secrets context is
  empty, environment secrets included. Docker workflows use no moon, no repository scripts and no
  cache.
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
- Run checks as explicit steps using ordinary moon commands. Keep generation and build preparation
  visible; dependent steps run only when their preparation succeeds. Independent checks continue
  after another check fails, without suppressing the failed step's or job's status.
- Drift checks follow regeneration in the job of the project they check; the embedded-build check
  runs in both the Go app and web UI jobs. They diff regenerated files and fail with an `::error`
  naming the local command to run.
- The `main` ruleset's code scanning rule waits for tool `zizmor` on every PR commit, so
  `pr-workflows-lint.yml` (PRs) and `main-security-scan.yml` (`main`) both upload zizmor SARIF with
  category `zizmor`. `pr-workflows-lint.yml` checks out the PR head SHA, so the upload goes to the
  head commit and `refs/pull/N/head`: GitHub regenerates the test merge commit with a new SHA, and
  results on the old merge SHA leave the rule waiting. In SARIF mode zizmor exits 0 on findings;
  the ruleset blocks on error-level or high alerts. A fork PR can't upload, so there
  `pr-workflows-lint.yml` runs zizmor without SARIF and fails on findings; its required `Security`
  check blocks the merge.
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
  `grep -n egress-policy .github/workflows/*.yml`. A job split off a blocked job may start with
  `block` and the parent's `allowed-endpoints`.
- List only what the audit runs show. The agent allows GitHub Actions and StepSecurity endpoints
  itself. Telemetry is disabled, not allowed: `telemetry` in `.prototools` and
  `.moon/workspace.yml`, `disable_telem: true` on every `codecov/codecov-action` step,
  `TRIVY_SKIP_VERSION_CHECK` and `TRIVY_DISABLE_TELEMETRY` on every `aquasecurity/trivy-action`
  step.
- A job running proto or moon under `block` sets job-level `env: PROTO_OFFLINE: "0"`, or proto's
  dropped connectivity probes break tool installs. Don't allow the probe hosts (`google.com`,
  `clients3.google.com`, `detectportal.firefox.com`).
- On a blocked connection, open the insights link the harden-runner step prints in the log and
  add the endpoint to that job's list.

## Naming

- Every workflow, job and step has an explicit `name:`, never a bare tool name or generic verb.
- Workflow: `Area: action` (`Repository: check`, `Workflows: lint`). Job: a short action without
  the area, since GitHub shows `Workflow / Job` (`Check Go app`, `Security report`; not `zizmor`). Step:
  the full action and subject (`Lint workflow syntax, expressions and shell scripts`, not `Run
  actionlint`).
- Job names are the required checks of the `main` ruleset; renaming a job means updating it. A
  reusable call shows as `caller job / called job`. Required: `Check Go app`, `Check web UI`,
  `Check repository`, `Docker image / Build and
  scan` and `Security`; the ruleset has no bypass actors.
- The ruleset matches checks by job name alone, so two jobs share a name only if they should
  count as one check.

## Reports and failure messages

- Actions passes report formats and paths to the same commands used locally. Keep check execution
  in workflow steps; reporting composites may share inline shell code, not hide check execution.
  Reports may live in `.ci/<project>/`; do not use files to relay annotations or control job status.
- Keep existing SARIF, JUnit, coverage and Codecov reporting. Security checks publish concrete
  package names, versions and links in the job log and Summary, plus machine-readable artifacts.
  Preserve govulncheck SARIF uploads with category `govulncheck`.
- Run each scan once, then convert its result for other report formats. Preserve the tool's exit
  status and distinguish blocking findings, excluded findings and tool errors. Missing or invalid
  required reports are errors, not successful scans; publication must not hide a check failure.
- Keep npm audit exclusions in Actions and pass them as arguments to `hump-tower:audit`; local
  audit checks all packages by default. Run `hump-tower:registry-check` separately. Preserve the
  existing exclusions, thresholds and review dates when changing reporting.

- Every failure says what is wrong and how to fix it, so the developer can act on the message
  alone. A bare non-zero exit or a raw tool dump is not enough.
- In a custom `run:` check, emit an error annotation before `exit 1`: point it at the file, title
  it with the problem, put the fix in the message. Print details (diff, values) to the log too.

  ```bash
  echo "::error file=go.mod,title=go.mod and go.sum are not tidy::Run 'go mod tidy' locally and commit the updated go.mod and go.sum."
  ```
