#!/usr/bin/env bash
# CI checks of the hump-tower web UI: ci.sh knip|biome|typecheck|test. Run by the hump-tower:ci-* moon tasks
# with the working directory apps/hump-tower.
#
# Reports (SARIF, coverage, JUnit) go to $MOON_WORKSPACE_ROOT/.ci/hump-tower. Annotations go to
# .ci/annotations/hump-tower-<check>.txt: moon prefixes task output, so workflow commands printed
# here would not become annotations. A failure also creates the marker
# .ci/failed/hump-tower-<check>. .github/workflows/check.yml prints those files, fails on the
# markers and reads the reports after 'moon ci'.

set -euo pipefail

main() {
    local check="${1:-}"
    ROOT="${MOON_WORKSPACE_ROOT:?MOON_WORKSPACE_ROOT is not set; run this through moon}"
    REPORTS="$ROOT/.ci/hump-tower"
    ANNOTATIONS="$ROOT/.ci/annotations"
    FAILED="$ROOT/.ci/failed"
    mkdir -p "$REPORTS" "$ANNOTATIONS"

    case "$check" in
        knip) knip ;;
        biome) biome ;;
        typecheck) typecheck ;;
        test) run_tests ;;
        *)
            echo "usage: ci.sh knip|biome|typecheck|test" >&2
            exit 2
            ;;
    esac
}

# knip finds unused dependencies, files and exports. It runs after hump-tower:build (a task dep),
# whose openapi-ts step creates src/shared/api/generated, which the imports knip follows resolve to.
knip() {
    local out="$REPORTS/knip.txt" status=0
    bun x --no-install knip --reporter github-actions >"$out" || status=$?
    # The findings are workflow commands only; the log shows them through the annotation file.
    grep -v '^::' "$out" || true
    findings knip "$out"
    if [[ "$status" -ne 0 ]]; then
        annotate knip "::error file=apps/hump-tower/package.json,title=Web UI has unused dependencies, files or exports::Run 'moon run hump-tower:knip' locally, then remove what it reports (or, for a false positive, adjust apps/hump-tower/knip.jsonc) and commit the changes."
    fi
    return "$status"
}

# biome ci runs the same lint, format and import order checks as 'moon run hump-tower:check',
# read-only. The default reporter keeps the text output, github feeds the annotations, sarif the
# job summary (sarif-summary strips $GITHUB_WORKSPACE from its absolute paths).
biome() {
    local out="$REPORTS/biome.txt" status=0
    bun x --no-install biome ci \
        --reporter=default \
        --reporter=github \
        --reporter=sarif \
        --reporter-file="$REPORTS/biome.sarif" >"$out" 2>&1 || status=$?
    grep -v '^::' "$out" || true
    findings biome "$out"
    if [[ "$status" -ne 0 ]]; then
        annotate biome "::error file=apps/hump-tower/biome.json,title=Web UI lint failed::Fix the biome findings listed in the log and in the job summary (formatting and import order are fixed by 'bun x biome check --write' in apps/hump-tower), then check locally with: moon run hump-tower:check"
    fi
    return "$status"
}

# typecheck runs the TypeScript compiler against tsconfig.json without emitting files.
typecheck() {
    local status=0
    bun x --no-install tsc --noEmit || status=$?
    if [[ "$status" -ne 0 ]]; then
        annotate typecheck "::error file=apps/hump-tower/tsconfig.json,title=Web UI type check failed::Fix the TypeScript errors listed in the log, then check locally with: moon run hump-tower:typecheck"
    fi
    return "$status"
}

# run_tests runs the unit tests with coverage. Coverage ignores (generated API client, test
# doubles) live in bunfig.toml [test], so a local 'bun test --coverage' reports the same total.
run_tests() {
    local status=0
    bun test \
        --coverage \
        --coverage-reporter=text \
        --coverage-reporter=lcov \
        --coverage-dir="$REPORTS/coverage" \
        --reporter=junit \
        --reporter-outfile="$REPORTS/junit.xml" || status=$?
    if [[ "$status" -ne 0 ]]; then
        annotate test "::error file=apps/hump-tower/package.json,title=Web UI unit tests failed::Fix the failing tests listed in the log, then check locally with: moon run hump-tower:test"
    fi
    return "$status"
}

# findings appends the workflow commands of a tool's output to the check's annotation file, with
# repository-relative paths. It writes nothing when there are none.
findings() {
    local check="$1" out="$2" commands
    commands="$(grep '^::' "$out" | repo_paths || true)"
    if [[ -n "$commands" ]]; then
        echo "$commands" >>"$ANNOTATIONS/hump-tower-${check}.txt"
    fi
}

# repo_paths makes the file= paths of workflow commands repository-relative: biome writes absolute
# paths, which lose the workspace root; knip writes paths relative to apps/hump-tower, which get
# the project prefix.
repo_paths() {
    sed -E \
        -e "s#file=${ROOT}/#file=#" \
        -e '/file=(apps\/|\/)/!s#file=#file=apps/hump-tower/#'
}

# annotate records a failed check: it appends the fix hint (a workflow command) to the check's
# annotation file and creates the check's marker in .ci/failed. The ci-* tasks are allowed to fail,
# so that one failure does not cancel the other checks; the workflow fails on the markers.
annotate() {
    local check="$1" line="$2"
    echo "$line" >>"$ANNOTATIONS/hump-tower-${check}.txt"
    mkdir -p "$FAILED"
    echo "$check" >"$FAILED/hump-tower-${check}"
}

main "$@"
