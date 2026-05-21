#!/bin/bash
# Download static ffmpeg/ffprobe for Linux from johnvansickle.com
set -euo pipefail

OUT_DIR="$(cd "$(dirname "$0")" && pwd)/bin"
ARCH="${1:-$(uname -m)}"
mkdir -p "$OUT_DIR"

BASE_URL="https://johnvansickle.com/ffmpeg/releases"
ARCHIVE="ffmpeg-release-${ARCH}-static.tar.xz"

echo "Downloading $ARCHIVE ..."
curl -L -o "/tmp/$ARCHIVE" "$BASE_URL/$ARCHIVE"

echo "Extracting ffmpeg and ffprobe ..."
tar -xf "/tmp/$ARCHIVE" -C "/tmp/"
cp "/tmp/ffmpeg-${ARCH}-static/ffmpeg" "$OUT_DIR/"
cp "/tmp/ffmpeg-${ARCH}-static/ffprobe" "$OUT_DIR/"
chmod +x "$OUT_DIR/ffmpeg" "$OUT_DIR/ffprobe"

rm -f "/tmp/$ARCHIVE"
rm -rf "/tmp/ffmpeg-${ARCH}-static"

echo "ffmpeg/ffprobe downloaded to: $OUT_DIR"
