#!/bin/bash
# Download minimal ffmpeg/ffprobe for macOS
set -euo pipefail

OUT_DIR="$(cd "$(dirname "$0")" && pwd)/bin"
mkdir -p "$OUT_DIR"

# Use Homebrew's ffmpeg if available
if command -v brew &>/dev/null; then
    echo "Installing ffmpeg via Homebrew..."
    brew install ffmpeg
    cp "$(brew --prefix ffmpeg)/bin/ffmpeg" "$OUT_DIR/"
    cp "$(brew --prefix ffmpeg)/bin/ffprobe" "$OUT_DIR/"
else
    # Download static build from evermeet.cx
    echo "Downloading ffmpeg static build..."
    curl -L -o "$OUT_DIR/ffmpeg.7z" "https://evermeet.cx/ffmpeg/ffmpeg-7.1.7z"
    curl -L -o "$OUT_DIR/ffprobe.7z" "https://evermeet.cx/ffmpeg/ffprobe-7.1.7z"
    7z x "$OUT_DIR/ffmpeg.7z" -o"$OUT_DIR/" -y
    7z x "$OUT_DIR/ffprobe.7z" -o"$OUT_DIR/" -y
    rm "$OUT_DIR/"*.7z
    chmod +x "$OUT_DIR/ffmpeg" "$OUT_DIR/ffprobe"
fi

echo "ffmpeg/ffprobe downloaded to: $OUT_DIR"
