#!/usr/bin/env bash
# CI checks of the hump-tower web UI: hump-tower-ci.sh knip|biome|typecheck|test|audit. Run by the
# hump-tower:ci-* moon tasks with the working directory apps/hump-tower.
#
# Reports (SARIF, coverage, JUnit) go to $MOON_WORKSPACE_ROOT/.ci/hump-tower. Annotations go to
# .ci/annotations/hump-tower-<check>.txt: moon prefixes task output, so workflow commands printed
# here would not become annotations. A failure also creates the marker
# .ci/failed/hump-tower-<check>. .github/workflows/ci-check.yml prints those files, fails on the
# markers and reads the reports after 'moon ci'.

set -euo pipefail

# Advisories bun audit skips: each has a fix only in a version published less than 7 days ago
# (root AGENTS.md, "Dependencies"). Remove an entry once its fix can be installed, on the date given.
AUDIT_IGNORES=(
    # happy-dom (dev, via @happy-dom/global-registrator, test DOM only): fixed in 21.0.4, published
    # 2026-10-20; also a major upgrade from 20.14.6. Revisit on 2026-10-28.
    GHSA-834v-8h66-3h79
    # fast-uri (dev, via ajv): fixed in 3.1.9, published 2026-10-19; 3.1.8 (root package.json
    # overrides) fixes the rest. Revisit on 2026-10-27.
    GHSA-w74x-v6f9-j3xg
    GHSA-xhq7-xmqj-769h
    GHSA-wjr9-j795-jxcr
    GHSA-333x-w4r6-6crp
)

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
        audit) audit ;;
        *)
            echo "usage: hump-tower-ci.sh knip|biome|typecheck|test|audit" >&2
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

# audit checks the packages of the root bun.lock, which covers every workspace, direct and
# transitive: GitHub's dependency graph sees only the direct npm dependencies, so Dependabot alerts
# miss the rest. bun audit fails on an advisory of any severity; the registry check fails on a
# package that does not come from registry.npmjs.org. Both run, whichever fails first.
audit() {
    local status=0
    vulnerabilities || status=$?
    registries || status=$?
    return "$status"
}

# vulnerabilities runs bun audit twice: the JSON run only feeds the report, the text run prints the
# advisories and fails the check. bun audit exits 1 when it finds advisories. 'moon run
# hump-tower:audit' runs bun audit without AUDIT_IGNORES, so it also lists the skipped advisories.
vulnerabilities() {
    local ignores=("${AUDIT_IGNORES[@]/#/--ignore=}")
    bun audit "${ignores[@]}" --json >"$REPORTS/audit.json" 2>/dev/null || true

    local status=0
    bun audit "${ignores[@]}" || status=$?
    if [[ "$status" -eq 1 ]]; then
        annotate audit "::error file=bun.lock,title=npm dependencies have known vulnerabilities::Upgrade the packages listed in the log to a fixed version that is at least 7 days old: raise a direct dependency in its package.json, or add an exact 'overrides' entry to the root package.json for a transitive one, run 'bun install' in the repository root and commit package.json and bun.lock. When no fixed version is 7 days old yet, add the advisory to AUDIT_IGNORES in scripts/hump-tower-ci.sh with the date to revisit it. Reproduce locally with: CI=true moon run hump-tower:ci-audit"
    elif [[ "$status" -ne 0 ]]; then
        annotate audit "::error file=bun.lock,title=npm vulnerability scan did not complete::bun audit exited with status ${status}; see the log for the cause. Reproduce locally with: CI=true moon run hump-tower:ci-audit"
    fi
    return "$status"
}

# registries fails when a package in bun.lock resolves from anywhere but registry.npmjs.org, or
# when a .npmrc or a registry setting in a bunfig.toml (root, apps/*, packages/*) could point bun
# at another registry. An npm entry in bun.lock is [name@version, registry, metadata, integrity],
# where an empty registry is the default one; the configuration check is what makes that default
# registry.npmjs.org. Workspace, link and file entries are local; git, GitHub and tarball URL
# entries are offenders.
registries() {
    local offenders status=0
    # The script is JavaScript: its $ and backticks are not meant for the shell.
    # shellcheck disable=SC2016
    offenders="$(
        bun -e '
            const lock = require(process.argv[1])
            for (const [key, [resolution, registry]] of Object.entries(lock.packages)) {
                const spec = resolution.slice(resolution.lastIndexOf("@") + 1)
                if (/^(workspace|link|file):/.test(spec)) continue
                const isNpm = /^\d/.test(spec) && typeof registry === "string"
                if (isNpm && (registry === "" || registry.startsWith("https://registry.npmjs.org/"))) continue
                console.log(`${key} -> ${resolution}${typeof registry === "string" && registry ? ` (${registry})` : ""}`)
            }
        ' "$ROOT/bun.lock"
    )" || status=$?
    if [[ "$status" -ne 0 ]]; then
        annotate audit "::error file=bun.lock,title=bun.lock could not be read::The registry check failed to parse bun.lock; see the log for the cause. Regenerate it with 'bun install' in the repository root and commit it."
        return "$status"
    fi

    local configs=() file
    shopt -s nullglob
    for file in "$ROOT"/{,apps/*/,packages/*/}{.npmrc,bunfig.toml}; do
        # The root paths have no wildcard, so nullglob keeps them whether or not they exist.
        [[ -f "$file" ]] || continue
        if [[ "$file" == */.npmrc ]] ||
            grep -q -E '^[[:space:]]*(registry[[:space:]]*=|\[install\.(registry|scopes)\])' "$file"; then
            configs+=("${file#"$ROOT"/}")
        fi
    done
    shopt -u nullglob

    if [[ -n "$offenders" ]]; then
        echo "Packages in bun.lock that do not resolve from https://registry.npmjs.org/:"
        echo "$offenders"
        annotate audit "::error file=bun.lock,title=bun.lock has packages from outside registry.npmjs.org::Replace these dependencies with versions published to https://registry.npmjs.org/, run 'bun install' in the repository root and commit package.json and bun.lock: $(echo "$offenders" | paste -sd ',' - | sed 's/,/, /g')"
        status=1
    fi
    if [[ "${#configs[@]}" -ne 0 ]]; then
        echo "Files that may set an npm registry: ${configs[*]}"
        annotate audit "::error file=bun.lock,title=npm registry configuration found::Remove the registry settings (or the .npmrc) from these files, so bun resolves every package from https://registry.npmjs.org/: ${configs[*]}"
        status=1
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
