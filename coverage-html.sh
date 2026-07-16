#!/bin/bash
set -e
TMPHTML="coverage_$$.html"
trap 'rm -f coverage.out coverage_filtered.out "$TMPHTML"' ERR

bash "build/$(uname -s | tr '[:upper:]' '[:lower:]')/download-ffmpeg.sh"

go test -coverprofile=coverage.out ./...
if [ -f .coverignore ]; then
  grep -v -f .coverignore coverage.out > coverage_filtered.out
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
