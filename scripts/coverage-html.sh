#!/bin/bash
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$SCRIPT_DIR/.."
TMPHTML="coverage_$$.html"
trap 'rm -f "$REPO_ROOT/src/coverage.out" "$REPO_ROOT/src/coverage_filtered.out" "$TMPHTML"' ERR

cd "$REPO_ROOT/src"
bash "build/$(uname -s | tr '[:upper:]' '[:lower:]')/download-ffmpeg.sh"

EXCLUDE="github.com/QuickOrBeDead/graftik-video-player/frontend"
go test -coverprofile=coverage.out $(go list ./... | grep -v "$EXCLUDE" | grep -v -E "$(sed '/^#/d;/^$/d;s|/\*$||' "$SCRIPT_DIR/.coverignore" 2>/dev/null | sed 's|\.|\\.|g;s|\*|.*|g;s|\?|.|g' | paste -sd'|' -)" 2>/dev/null || go list ./... | grep -v "$EXCLUDE")
if [ -f "$SCRIPT_DIR/.coverignore" ]; then
  sed '/^#/d;/^$/d;s|\.|\\.|g;s|\*\*|____GLOBSTAR____|g;s|\*|[^/]*|g;s|\?|[^/]|g;s|____GLOBSTAR____|.*|g' \
    "$SCRIPT_DIR/.coverignore" | grep -v -f - coverage.out > coverage_filtered.out
  go tool cover -html=coverage_filtered.out -o "$TMPHTML"
else
  go tool cover -html=coverage.out -o "$TMPHTML"
fi
rm -f coverage.out coverage_filtered.out

google-chrome "$TMPHTML" &
sleep 2
while pgrep -f "$TMPHTML" > /dev/null 2>&1; do
  sleep 1
done
rm -f "$TMPHTML"
