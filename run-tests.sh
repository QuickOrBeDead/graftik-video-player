#!/bin/bash
# Prepare ffmpeg and run Go tests
# Usage: ./run-tests.sh [--short]
set -euo pipefail

SHORT=false
case "${1:-}" in
  --short) SHORT=true   ;;
  -s)      SHORT=true   ;;
  "")                   ;;
  *) echo "Usage: $0 [--short]" >&2; exit 1 ;;
esac

cd "$(dirname "$0")"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux)  SCRIPT="build/linux/download-ffmpeg.sh"   SCRIPT_INTERP="bash" ;;
  darwin) SCRIPT="build/darwin/download-ffmpeg.sh"  SCRIPT_INTERP="bash" ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

if $SHORT; then
  echo "=== Running Go tests (short mode) ==="
  exec go test -short ./...
fi

echo "=== Downloading ffmpeg ==="
"$SCRIPT_INTERP" "$SCRIPT"

echo ""
echo "=== Running Go tests ==="
exec go test ./...
