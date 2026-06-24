#!/bin/bash
set -euo pipefail

OUT_DIR="$(cd "$(dirname "$0")" && pwd)/bin"
ARCH="${1:-$(uname -m)}"

case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64"  ;;
  armv7l|armhf)  ARCH="armhf"  ;;
  armel)         ARCH="armel"  ;;
  i686|i386)     ARCH="i686"   ;;
  *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

mkdir -p "$OUT_DIR"

FFMPEG_VERSION="release"
ARCHIVE="ffmpeg-${FFMPEG_VERSION}-${ARCH}-static.tar.xz"
BASE_URL="https://johnvansickle.com/ffmpeg/releases"

echo "Downloading $ARCHIVE ..."
curl -L -o "/tmp/$ARCHIVE" "$BASE_URL/$ARCHIVE"

echo "Extracting ffmpeg and ffprobe ..."
tar -xf "/tmp/$ARCHIVE" -C "/tmp/"
EXTRACT_DIR="$(find /tmp -mindepth 1 -maxdepth 1 -type d -name 'ffmpeg-*' | head -1)"
cp "$EXTRACT_DIR/ffmpeg" "$OUT_DIR/"
cp "$EXTRACT_DIR/ffprobe" "$OUT_DIR/"
chmod +x "$OUT_DIR/ffmpeg" "$OUT_DIR/ffprobe"

rm -f "/tmp/$ARCHIVE"
rm -rf "$EXTRACT_DIR"

echo "ffmpeg/ffprobe downloaded to: $OUT_DIR"
