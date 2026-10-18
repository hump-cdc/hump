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
- actionlint is pinned as a Go tool in its own module, `tools/actionlint/go.mod`. It can't live in
  the root `go.mod` because its dependencies conflict with golangci-lint's. Run it as
  `go tool -modfile=tools/actionlint/go.mod actionlint`.

## Naming

- Every workflow, job and step has an explicit `name:`. Don't use the tool name or a generic verb
  on its own.
- Workflow: `Area: action`, a short area, a colon and a short lowercase action: `Go: lint`,
  `Web: test`, `Workflows: security report`; not `GitHub Actions workflows lint`.
- Job: a short action without the area, because GitHub shows checks as `Workflow / Job`
  (`Go: lint / Lint`): `Lint`, `Build and test`, `Security`; not `zizmor`.
- Step: the full action and its subject: `Lint workflow syntax, expressions and shell scripts`,
  not `Run actionlint`.
- Job names are the status-check names that the branch ruleset on `main` lists as required
  checks. Renaming a job requires updating the ruleset.
- The ruleset matches a required check by job name alone. This project deliberately has two jobs
  named `Lint` (the Go and Web UI lint workflows).

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
