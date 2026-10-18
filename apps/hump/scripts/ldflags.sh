#!/bin/sh
# Prints the go build -ldflags that stamp pkg/version: VERSION (default 0.0.0),
# the short commit and the UTC build date.
set -eu

commit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)
date=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo "-s -w -X hump/pkg/version.Version=${VERSION:-0.0.0} -X hump/pkg/version.Commit=${commit} -X hump/pkg/version.Date=${date}"
