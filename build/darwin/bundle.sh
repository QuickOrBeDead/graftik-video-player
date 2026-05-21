#!/bin/bash
# Create macOS .app bundle with ffmpeg/ffprobe included
# Usage: ./bundle.sh [path-to-binary]
set -euo pipefail

APP_NAME="GraftikVideoPlayer"
BINARY="${1:-../bin/$APP_NAME}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BUNDLE_DIR="$BUILD_DIR/$APP_NAME.app"
CONTENTS_DIR="$BUNDLE_DIR/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"

echo "Creating .app bundle at $BUNDLE_DIR ..."

mkdir -p "$MACOS_DIR/bin"
mkdir -p "$RESOURCES_DIR"

# Copy binary
cp "$BINARY" "$MACOS_DIR/$APP_NAME"

# Copy ffmpeg/ffprobe into bin/
if [ -d "$SCRIPT_DIR/bin" ]; then
    cp "$SCRIPT_DIR/bin/ffmpeg" "$MACOS_DIR/bin/" 2>/dev/null || true
    cp "$SCRIPT_DIR/bin/ffprobe" "$MACOS_DIR/bin/" 2>/dev/null || true
    chmod +x "$MACOS_DIR/bin/ffmpeg" "$MACOS_DIR/bin/ffprobe" 2>/dev/null || true
fi

# Copy icon
cp "$BUILD_DIR/appicon.png" "$RESOURCES_DIR/icon.png" 2>/dev/null || true

# Create Info.plist
cat > "$CONTENTS_DIR/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>$APP_NAME</string>
    <key>CFBundleIdentifier</key>
    <string>com.graftik.videoplayer</string>
    <key>CFBundleName</key>
    <string>Graftik Video Player</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleIconFile</key>
    <string>icon</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSRequiresAquaSystemAppearance</key>
    <false/>
</dict>
</plist>
EOF

echo "Bundle created at: $BUNDLE_DIR"
