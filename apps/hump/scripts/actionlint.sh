#!/usr/bin/env bash
# root:actionlint and root:ci-actionlint. Lints .github/workflows with actionlint, pinned as a Go
# tool in the separate module apps/hump/tools/actionlint. Runs from the workspace root.
#
# The module is outside go.work (its dependencies conflict with golangci-lint's) and -modfile is
# rejected in workspace mode, so it runs in its own directory with GOWORK=off. actionlint finds
# .github by walking up to the repository root.
#
# In GitHub Actions a failure also writes a fix hint to .ci/annotations/root-actionlint.txt and
# the marker .ci/failed/root-actionlint: moon prefixes task output, so workflow commands printed
# here would not become annotations, and ci-actionlint is allowed to fail so that the other
# checks keep running. .github/workflows/ci-check.yml reads both after 'moon ci'.

set -euo pipefail

status=0
GOWORK=off go -C apps/hump/tools/actionlint tool actionlint || status=$?

if [[ "$status" -ne 0 && "${GITHUB_ACTIONS:-}" == "true" ]]; then
    ci="${MOON_WORKSPACE_ROOT:?MOON_WORKSPACE_ROOT is not set; run this through moon}/.ci"
    mkdir -p "$ci/annotations" "$ci/failed"
    echo "::error file=.github/workflows,title=Workflow lint failed::Fix the actionlint findings listed in the log, then check locally with: moon run root:actionlint" \
        >>"$ci/annotations/root-actionlint.txt"
    echo actionlint >"$ci/failed/root-actionlint"
fi

exit "$status"
