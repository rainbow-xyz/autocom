#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PACKAGE_NAME="autocom-skills"
DIST_DIR="$ROOT_DIR/dist"
PACKAGE_DIR="$DIST_DIR/$PACKAGE_NAME"

rm -rf "$PACKAGE_DIR" "$DIST_DIR/$PACKAGE_NAME.tar.gz"

(
  cd "$ROOT_DIR"
  python3 -m unittest discover -s skills/image-generation/providers/zlhub/scripts -p 'test_*.py'
  python3 -m unittest discover -s skills/video-generation/providers/zlhub/scripts -p 'test_*.py'
)

find "$ROOT_DIR/skills/image-generation" "$ROOT_DIR/skills/video-generation" -type d -name __pycache__ -prune -exec rm -rf {} +

mkdir -p "$PACKAGE_DIR/skills" "$PACKAGE_DIR/docs"
cp -R "$ROOT_DIR/skills/image-generation" "$PACKAGE_DIR/skills/image-generation"
cp -R "$ROOT_DIR/skills/video-generation" "$PACKAGE_DIR/skills/video-generation"
cp -R "$ROOT_DIR/docs/paperclip" "$PACKAGE_DIR/docs/paperclip"
chmod +x "$PACKAGE_DIR/skills/image-generation/providers/zlhub/scripts/zlhub_cli.py"
chmod +x "$PACKAGE_DIR/skills/video-generation/providers/zlhub/scripts/zlhub_cli.py"
find "$PACKAGE_DIR" -type d -name __pycache__ -prune -exec rm -rf {} +

tar -czf "$DIST_DIR/$PACKAGE_NAME.tar.gz" -C "$DIST_DIR" "$PACKAGE_NAME"
shasum -a 256 "$DIST_DIR/$PACKAGE_NAME.tar.gz"
