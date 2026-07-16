#!/bin/bash
# Download ffmpeg/ffprobe 8.1.2 for macOS from GitHub Releases
set -euo pipefail

OUT_DIR="$(cd "$(dirname "$0")" && pwd)/bin"
mkdir -p "$OUT_DIR"

MARKER="/tmp/ffmpeg-8.1.2-darwin.verified"
if [[ -f "$MARKER" ]] && [[ -f "${OUT_DIR}/ffmpeg" ]]; then
  echo "ffmpeg already present and verified at ${OUT_DIR}"
  exit 0
fi

EXPECTED_HASH="1ded0195068e24d9ec961ce9d8ea41944cd3a51ecccf270599c4b6485c24edfc"

# Use Homebrew's ffmpeg if available
if command -v brew &>/dev/null; then
    echo "Installing ffmpeg via Homebrew..."
    brew install ffmpeg
    cp "$(brew --prefix ffmpeg)/bin/ffmpeg" "$OUT_DIR/"
    cp "$(brew --prefix ffmpeg)/bin/ffprobe" "$OUT_DIR/"
    touch "$MARKER"
    echo "ffmpeg/ffprobe downloaded to: $OUT_DIR"
    exit 0
fi

OWNER="QuickOrBeDead"
REPO="graftik-video-player"
TAG="ffmpeg-8.1.2"
ARCHIVE="darwin-amd64-${TAG}.tar.gz"
EXPECTED_HASH="1ded0195068e24d9ec961ce9d8ea41944cd3a51ecccf270599c4b6485c24edfc"
BASE_URL="https://github.com/${OWNER}/${REPO}/releases/download/${TAG}"

echo "Downloading ${ARCHIVE} ..."
curl -L -o "/tmp/${ARCHIVE}" "${BASE_URL}/${ARCHIVE}"

echo "Verifying integrity ..."
echo "${EXPECTED_HASH}  /tmp/${ARCHIVE}" | sha256sum -c -

echo "Extracting ffmpeg and ffprobe ..."
tar -xzf "/tmp/${ARCHIVE}" -C "${OUT_DIR}/"
chmod +x "${OUT_DIR}/ffmpeg" "${OUT_DIR}/ffprobe"

rm -f "/tmp/${ARCHIVE}"
touch "$MARKER"

echo "ffmpeg/ffprobe downloaded to: ${OUT_DIR}"
