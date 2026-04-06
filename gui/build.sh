#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_DIR="$SCRIPT_DIR/build/bin/Crunch.app/Contents/MacOS"
FFMPEG_DIR="$SCRIPT_DIR/build/ffmpeg-cache"

echo "=== Crunch Build ==="

# Step 1: Build the Wails app
echo "[1/3] Building Wails app..."
cd "$SCRIPT_DIR"
wails build 2>&1 | grep -E "Built|ERROR" || true

if [ ! -f "$APP_DIR/Crunch" ]; then
    echo "ERROR: Wails build failed"
    exit 1
fi

# Step 2: Download static ffmpeg/ffprobe if not cached
mkdir -p "$FFMPEG_DIR"

if [ ! -f "$FFMPEG_DIR/ffmpeg" ] || [ ! -f "$FFMPEG_DIR/ffprobe" ]; then
    echo "[2/3] Downloading static ffmpeg 8.1 + ffprobe 8.1..."

    case "$(uname -m)" in
        arm64)
            FFMPEG_URL="https://www.osxexperts.net/ffmpeg81arm.zip"
            FFPROBE_URL="https://www.osxexperts.net/ffprobe81arm.zip"
            ;;
        x86_64)
            FFMPEG_URL="https://www.osxexperts.net/ffmpeg81intel.zip"
            FFPROBE_URL="https://www.osxexperts.net/ffprobe81intel.zip"
            ;;
        *)
            echo "ERROR: Unsupported architecture $(uname -m)"
            exit 1
            ;;
    esac

    if [ ! -f "$FFMPEG_DIR/ffmpeg" ]; then
        echo "  Downloading ffmpeg..."
        curl -L -o /tmp/crunch-ffmpeg.zip "$FFMPEG_URL"
        unzip -o -j /tmp/crunch-ffmpeg.zip -d "$FFMPEG_DIR" 2>/dev/null || true
        rm -f /tmp/crunch-ffmpeg.zip
        chmod +x "$FFMPEG_DIR/ffmpeg"
    fi

    if [ ! -f "$FFMPEG_DIR/ffprobe" ]; then
        echo "  Downloading ffprobe..."
        curl -L -o /tmp/crunch-ffprobe.zip "$FFPROBE_URL"
        unzip -o -j /tmp/crunch-ffprobe.zip -d "$FFMPEG_DIR" 2>/dev/null || true
        rm -f /tmp/crunch-ffprobe.zip
        chmod +x "$FFMPEG_DIR/ffprobe"
    fi
else
    echo "[2/3] Using cached ffmpeg binaries"
fi

# Verify downloads
if [ ! -f "$FFMPEG_DIR/ffmpeg" ] || [ ! -f "$FFMPEG_DIR/ffprobe" ]; then
    echo "ERROR: ffmpeg binaries not found after download"
    exit 1
fi

# Step 3: Bundle into app
echo "[3/3] Bundling ffmpeg into Crunch.app..."
cp "$FFMPEG_DIR/ffmpeg" "$APP_DIR/ffmpeg"
cp "$FFMPEG_DIR/ffprobe" "$APP_DIR/ffprobe"
chmod +x "$APP_DIR/ffmpeg" "$APP_DIR/ffprobe"

# Report
APP_SIZE=$(du -sh "$SCRIPT_DIR/build/bin/Crunch.app" | cut -f1)
echo ""
echo "=== Build complete ==="
echo "App: $SCRIPT_DIR/build/bin/Crunch.app ($APP_SIZE)"
"$APP_DIR/ffmpeg" -version 2>&1 | head -1
"$APP_DIR/ffprobe" -version 2>&1 | head -1
