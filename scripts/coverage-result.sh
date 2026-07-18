#!/bin/bash
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$SCRIPT_DIR/.."
trap 'rm -f "$REPO_ROOT/src/coverage.out" "$REPO_ROOT/src/coverage_filtered.out"' ERR

cd "$REPO_ROOT/src"
bash "build/$(uname -s | tr '[:upper:]' '[:lower:]')/download-ffmpeg.sh"

EXCLUDE="github.com/QuickOrBeDead/graftik-video-player/frontend"
go test -coverprofile=coverage.out $(go list ./... | grep -v "$EXCLUDE" | grep -v -E "$(sed '/^#/d;/^$/d;s|/\*$||' "$SCRIPT_DIR/.coverignore" 2>/dev/null | sed 's|\.|\\.|g;s|\*|.*|g;s|\?|.|g' | paste -sd'|' -)" 2>/dev/null || go list ./... | grep -v "$EXCLUDE")
if [ -f "$SCRIPT_DIR/.coverignore" ]; then
  sed '/^#/d;/^$/d;s|\.|\\.|g;s|\*\*|____GLOBSTAR____|g;s|\*|[^/]*|g;s|\?|[^/]|g;s|____GLOBSTAR____|.*|g' \
    "$SCRIPT_DIR/.coverignore" | grep -v -f - coverage.out > coverage_filtered.out
  go tool cover -func=coverage_filtered.out | tail -n 1
else
  go tool cover -func=coverage.out | tail -n 1
fi
rm -f coverage.out coverage_filtered.out
