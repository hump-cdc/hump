#!/usr/bin/env bash
# Runs the tests in <dir> with coverage and opens the HTML report.
# Usage: cover.sh <dir> [go test args...]
set -euo pipefail

if [[ $# -lt 1 ]]; then
    echo "usage: $0 <dir> [go test args...]" >&2
    exit 2
fi

cd "$1"
shift

trap 'rm -f cover.out' EXIT

report="${TMPDIR:-/tmp}/hump-cover.html"

go tool gotestsum --format testdox -- -coverprofile=cover.out "$@"
go tool gocov convert cover.out | go tool gocov-html > "$report"

case "$(uname -s)" in
    Darwin) open "$report" ;;
    # MSYS_NO_PATHCONV keeps Git Bash from rewriting /c into a drive path.
    MINGW* | MSYS* | CYGWIN*) MSYS_NO_PATHCONV=1 cmd /c start "" "$(cygpath -w "$report")" ;;
    *) xdg-open "$report" ;;
esac
