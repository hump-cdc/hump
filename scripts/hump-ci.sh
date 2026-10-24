#!/usr/bin/env bash
# CI checks of the hump Go module: hump-ci.sh lint|vulncheck|test|dep-age. Run by the hump:ci-*
# moon tasks with the working directory apps/hump.
#
# Reports (SARIF, coverage, JUnit) go to $MOON_WORKSPACE_ROOT/.ci/hump. A failure appends a fix
# hint as a workflow command to .ci/annotations/hump-<check>.txt (moon prefixes task output, so
# workflow commands printed here would not become annotations) and creates the marker
# .ci/failed/hump-<check>. .github/workflows/ci-check.yml prints those files, fails on the markers and
# reads the reports after 'moon ci'; .github/workflows/main-security-scan.yml does the same for vulncheck.

set -euo pipefail

# Minimum age of a Go module version, matching the Dependabot cooldown in .github/dependabot.yml.
MIN_AGE_DAYS=7

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
        dep-age) dep_age ;;
        *)
            echo "usage: hump-ci.sh lint|vulncheck|test|dep-age" >&2
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

# vulncheck builds govulncheck from the separate tool module tools/govulncheck (outside go.work,
# so GOWORK=off) rather than starting it with go run, which turns every non-zero exit into 1 and
# would hide exit code 3 (the code calls a vulnerable symbol). The SARIF run always exits 0 and only
# feeds the job summary and code scanning, with its paths made repository-relative; the text run
# fails the check and prints the call traces.
vulncheck() {
    local govulncheck="$REPORTS/bin/govulncheck"
    if ! GOWORK=off go -C tools/govulncheck build -o "$govulncheck" golang.org/x/vuln/cmd/govulncheck; then
        annotate vulncheck "::error file=apps/hump/tools/govulncheck/go.mod,title=Vulnerability scan did not complete::Building govulncheck from apps/hump/tools/govulncheck failed; see the log for the cause. Reproduce locally with: CI=true moon run hump:ci-vulncheck"
        return 1
    fi
    "$govulncheck" -format sarif ./... >"$REPORTS/govulncheck.sarif"
    local rewritten=0
    relative_sarif_paths "$REPORTS/govulncheck.sarif" || rewritten=1

    local status=0
    "$govulncheck" ./... || status=$?
    if [[ "$status" -eq 0 ]]; then
        status="$rewritten"
    elif [[ "$status" -eq 3 ]]; then
        annotate vulncheck "::error file=apps/hump/go.mod,title=Code calls known vulnerabilities::Upgrade the modules or the Go version in apps/hump/go.mod to the 'Fixed in' versions from the log (go get <module>@<version> or go mod edit -go=<version>, then moon run hump:tidy) and commit go.mod and go.sum. Reproduce locally with: CI=true moon run hump:ci-vulncheck"
    elif [[ "$status" -ne 0 ]]; then
        annotate vulncheck "::error file=apps/hump/go.mod,title=Vulnerability scan did not complete::govulncheck exited with status ${status}; see the log for the cause. Reproduce locally with: CI=true moon run hump:ci-vulncheck"
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

# dep_age fails when a module version was published less than MIN_AGE_DAYS days ago: every module
# in the build list (direct and transitive) of apps/hump/go.mod and of the separate tool modules
# tools/actionlint/go.mod and tools/govulncheck/go.mod. GOWORK=off gives each go.mod its own build
# list instead of the go.work one. Publish times come from the module proxy; modules without one
# (replaced by a local directory) are skipped. A version exactly MIN_AGE_DAYS old passes, so
# Dependabot's own PRs, opened after its cooldown, pass too.
dep_age() {
    local format="2006-01-02T15:04:05Z"
    local template="{{if and (not .Main) .Time}}{{.Path}} {{.Version}} {{.Time.UTC.Format \"${format}\"}} {{(.Time.AddDate 0 0 ${MIN_AGE_DAYS}).UTC.Format \"${format}\"}}{{end}}"
    local cutoff
    cutoff="$(cutoff_time)"
    echo "Module versions published after ${cutoff} are younger than ${MIN_AGE_DAYS} days."

    local status=0
    check_ages apps/hump/go.mod "$cutoff" \
        "list versions with 'go list -m -versions <module>', check the publish time (Time) of one with 'go list -m -json <module>@<version>', then in apps/hump run 'go get <module>@<version>' and 'moon run hump:tidy' and commit go.mod and go.sum" \
        env GOWORK=off go list -m -f "$template" all || status=1
    check_ages apps/hump/tools/actionlint/go.mod "$cutoff" \
        "list versions with 'go list -m -versions <module>', check the publish time (Time) of one with 'go list -m -json <module>@<version>', then in apps/hump/tools/actionlint run 'GOWORK=off go get <module>@<version>' and 'moon run hump:tidy' and commit go.mod and go.sum" \
        env GOWORK=off go -C tools/actionlint list -m -f "$template" all || status=1
    check_ages apps/hump/tools/govulncheck/go.mod "$cutoff" \
        "list versions with 'go list -m -versions <module>', check the publish time (Time) of one with 'go list -m -json <module>@<version>', then in apps/hump/tools/govulncheck run 'GOWORK=off go get <module>@<version>' and 'moon run hump:tidy' and commit go.mod and go.sum" \
        env GOWORK=off go -C tools/govulncheck list -m -f "$template" all || status=1
    return "$status"
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

# check_ages runs a 'go list -m' command that prints '<module> <version> <published> <allowed>'
# lines and annotates every module published after the cutoff against file. fix is the remedy for
# file, with <module> and <version> as placeholders. Times are fixed-width RFC 3339 UTC, so they
# compare as strings.
check_ages() {
    local file="$1" cutoff="$2" fix="$3"
    shift 3
    local listing
    if ! listing="$("$@")"; then
        annotate dep-age "::error file=${file},title=Go module age check did not complete::'go list -m' failed for ${file}; see the log for the cause. Reproduce locally with: CI=true moon run hump:ci-dep-age"
        return 1
    fi

    local young=0 module version published allowed
    while read -r module version published allowed; do
        if [[ -z "$module" || ! "$published" > "$cutoff" ]]; then
            continue
        fi
        echo "${file}: ${module} ${version} was published ${published}, allowed from ${allowed}"
        local hint="${fix//<module>/${module}}"
        annotate dep-age "::error file=${file},title=Go module younger than ${MIN_AGE_DAYS} days::${module} ${version} was published ${published} and is allowed from ${allowed} (${MIN_AGE_DAYS}-day minimum release age; see .github/dependabot.yml). Pick an older version: ${hint}. Or wait until ${allowed}. Reproduce locally with: CI=true moon run hump:ci-dep-age"
        young=1
    done <<<"$listing"
    return "$young"
}

# relative_sarif_paths makes the paths in a govulncheck SARIF report repository-relative. govulncheck
# writes them relative to the module (go.mod, internal/...) with the base %SRCROOT%; code scanning
# and the links in the job summary resolve them from the repository root, so they get the
# apps/hump/ prefix. %GOROOT% paths (standard library frames) are left as they are.
relative_sarif_paths() {
    local file="$1" tmp
    tmp="$(mktemp)"
    if ! jq '(.. | objects | select(.uriBaseId? == "%SRCROOT%" and (.uri | type) == "string" and (.uri | startswith("apps/hump/") | not)) | .uri) |= "apps/hump/" + .' "$file" >"$tmp"; then
        rm -f "$tmp"
        annotate vulncheck "::error file=scripts/hump-ci.sh,title=Go vulnerability report could not be rewritten::jq failed to make the paths in .ci/hump/govulncheck.sarif repository-relative (invalid JSON or jq missing); see the log for the cause. Reproduce locally with: CI=true moon run hump:ci-vulncheck"
        return 1
    fi
    mv "$tmp" "$file"
}

# cutoff_time prints the time MIN_AGE_DAYS days ago in RFC 3339 UTC. date -d is GNU (CI), date -r
# is BSD (macOS).
cutoff_time() {
    local cutoff=$(($(date -u +%s) - MIN_AGE_DAYS * 86400))
    date -u -d "@${cutoff}" +%FT%TZ 2>/dev/null || date -u -r "$cutoff" +%FT%TZ
}

main "$@"
