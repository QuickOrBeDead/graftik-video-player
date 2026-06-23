#!/bin/bash
# Create Linux AppImage/tarball with ffmpeg/ffprobe included
# Usage: ./package.sh [path-to-binary]
set -euo pipefail

APP_NAME="graftik-video-player"
BINARY="${1:-../bin/$APP_NAME}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
STAGING_DIR="$BUILD_DIR/staging"
TARBALL="$BUILD_DIR/$APP_NAME-linux-amd64.tar.gz"

echo "Creating staging directory ..."
rm -rf "$STAGING_DIR"
mkdir -p "$STAGING_DIR/bin"

# Copy binary
cp "$BINARY" "$STAGING_DIR/$APP_NAME"
chmod +x "$STAGING_DIR/$APP_NAME"

# Copy ffmpeg/ffprobe into bin/
if [ -d "$SCRIPT_DIR/bin" ]; then
    cp "$SCRIPT_DIR/bin/ffmpeg" "$STAGING_DIR/bin/"
    cp "$SCRIPT_DIR/bin/ffprobe" "$STAGING_DIR/bin/"
    chmod +x "$STAGING_DIR/bin/ffmpeg" "$STAGING_DIR/bin/ffprobe"
fi

# Create launcher script
cat > "$STAGING_DIR/$APP_NAME.sh" << 'LAUNCHER'
#!/bin/bash
DIR="$(cd "$(dirname "$0")" && pwd)"
"$DIR/graftik-video-player" "$@"
LAUNCHER
chmod +x "$STAGING_DIR/$APP_NAME.sh"

# Create .desktop file
cat > "$STAGING_DIR/$APP_NAME.desktop" << EOF
[Desktop Entry]
Name=Graftik Video Player
Exec=$APP_NAME.sh
Icon=$APP_NAME
Type=Application
Categories=AudioVideo;Player;
EOF

# Package tarball
echo "Creating $TARBALL ..."
tar -czf "$TARBALL" -C "$STAGING_DIR" .

echo "Package created: $TARBALL"
echo ""
echo "For AppImage, use appimagetool on the staging directory:"
echo "  appimagetool $STAGING_DIR"
