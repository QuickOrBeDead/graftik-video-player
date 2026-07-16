#!/bin/bash
# Download ffmpeg/ffprobe 8.1.2 for Linux from GitHub Releases
set -euo pipefail

OUT_DIR="$(cd "$(dirname "$0")" && pwd)/bin"
mkdir -p "$OUT_DIR"

MARKER="/tmp/ffmpeg-8.1.2-linux.verified"
if [[ -f "$MARKER" ]] && [[ -f "${OUT_DIR}/ffmpeg" ]]; then
  echo "ffmpeg already present and verified at ${OUT_DIR}"
  exit 0
fi

OWNER="QuickOrBeDead"
REPO="graftik-video-player"
TAG="ffmpeg-8.1.2"
ARCHIVE="linux-amd64-${TAG}.tar.gz"
EXPECTED_HASH="cc6aab4abb4a57aa27a765e5ad99caf9ce13d47bc156ae815f5122991657632b"
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
