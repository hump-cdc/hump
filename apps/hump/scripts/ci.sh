#!/usr/bin/env bash
# CI checks of the hump Go module: ci.sh lint|vulncheck|test. Run by the hump:ci-* moon tasks with
# the working directory apps/hump.
#
# Reports (SARIF, coverage, JUnit) go to $MOON_WORKSPACE_ROOT/.ci/hump. A failure appends a fix
# hint as a workflow command to .ci/annotations/hump-<check>.txt (moon prefixes task output, so
# workflow commands printed here would not become annotations) and creates the marker
# .ci/failed/hump-<check>. .github/workflows/check.yml prints those files, fails on the markers and
# reads the reports after 'moon ci'.

set -euo pipefail

# Dependabot does not bump this version.
GOVULNCHECK_VERSION="v1.9.0"

main() {
    local check="${1:-}"
    local root="${MOON_WORKSPACE_ROOT:?MOON_WORKSPACE_ROOT is not set; run this through moon}"
    REPORTS="$root/.ci/hump"
    ANNOTATIONS="$root/.ci/annotations"
    FAILED="$root/.ci/failed"
    mkdir -p "$REPORTS" "$ANNOTATIONS"

    case "$check" in
        lint) lint ;;
        vulncheck) vulncheck ;;
        test) run_tests ;;
        *)
            echo "usage: ci.sh lint|vulncheck|test" >&2
            exit 2
            ;;
    esac
}

# lint runs golangci-lint (version from go.mod) against .golangci.yml. A SARIF output replaces the
# default text output, so the text output is kept explicitly; --path-prefix makes the paths in
# both repository-relative.
lint() {
    local status=0
    go tool golangci-lint run \
        --path-prefix=apps/hump \
        --output.text.path=stdout \
        --output.sarif.path="$REPORTS/golangci-lint.sarif" || status=$?
    if [[ "$status" -ne 0 ]]; then
        annotate lint "::error file=apps/hump/.golangci.yml,title=Go lint failed::Fix the golangci-lint findings listed in the log and in the job summary (many are fixed by 'moon run hump:lint-fix'), then check locally with: moon run hump:lint"
    fi
    return "$status"
}

# vulncheck installs govulncheck rather than starting it with go run, which turns every non-zero
# exit into 1 and would hide exit code 3 (the code calls a vulnerable symbol). The SARIF run always
# exits 0 and only feeds the job summary; the text run fails the check and prints the call traces.
vulncheck() {
    local package="golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}"
    local govulncheck="$REPORTS/bin/govulncheck"
    GOBIN="$REPORTS/bin" go install "$package"
    "$govulncheck" -format sarif ./... >"$REPORTS/govulncheck.sarif"

    local status=0
    "$govulncheck" ./... || status=$?
    if [[ "$status" -eq 3 ]]; then
        annotate vulncheck "::error file=apps/hump/go.mod,title=Code calls known vulnerabilities::Upgrade the modules or the Go version in apps/hump/go.mod to the 'Fixed in' versions from the log (go get <module>@<version> or go mod edit -go=<version>, then moon run hump:tidy) and commit go.mod and go.sum. Reproduce locally in apps/hump with: go run ${package} ./..."
    elif [[ "$status" -ne 0 ]]; then
        annotate vulncheck "::error file=apps/hump/go.mod,title=Vulnerability scan did not complete::govulncheck exited with status ${status}; see the log for the cause. Reproduce locally in apps/hump with: go run ${package} ./..."
    fi
    return "$status"
}

# run_tests runs the unit tests with the race detector and coverage. Integration tests carry the
# 'integration' build tag and are left out because -tags is not passed. -race needs cgo, which the
# hump:ci-test task enables. -coverpkg=./... counts lines a test in one package covers in another,
# the same as 'moon run hump:test-cover'.
run_tests() {
    local status=0
    go tool gotestsum \
        --format testname \
        --junitfile "$REPORTS/junit.xml" \
        -- \
        -race \
        -covermode=atomic \
        -coverpkg=./... \
        -coverprofile="$REPORTS/cover.out" \
        ./... || status=$?
    if [[ "$status" -ne 0 ]]; then
        annotate test "::error file=apps/hump/go.mod,title=Go unit tests failed::Fix the failing tests or data races listed in the log, then check locally with: moon run hump:test (add -race in apps/hump with: CGO_ENABLED=1 go tool gotestsum -- -race ./...)"
        return "$status"
    fi

    # cmd/openapi only writes the OpenAPI spec at build time, so it is dropped from the profile;
    # codecov.yml ignores the same path. The filtered profile feeds both the job summary and
    # Codecov, so the two count the same code.
    grep -v '^hump/cmd/openapi/' "$REPORTS/cover.out" >"$REPORTS/coverage.out"
}

# annotate records a failed check: it appends the fix hint (a workflow command) to the check's
# annotation file and creates the check's marker in .ci/failed. The ci-* tasks are allowed to fail,
# so that one failure does not cancel the other checks; the workflow fails on the markers.
annotate() {
    local check="$1" line="$2"
    echo "$line" >>"$ANNOTATIONS/hump-${check}.txt"
    mkdir -p "$FAILED"
    echo "$check" >"$FAILED/hump-${check}"
}

main "$@"
