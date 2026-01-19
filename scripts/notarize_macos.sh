#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <path-to-macos-binary>" >&2
  echo "Example: $0 dist/autocodex_darwin_arm64_v1/autocodex" >&2
  exit 1
fi

BIN="$1"
ZIP="${BIN}.zip"

if [ ! -f "$BIN" ]; then
  echo "error: binary not found: $BIN" >&2
  exit 1
fi

if [ -z "${MACOS_SIGN_IDENTITY:-}" ]; then
  echo "error: MACOS_SIGN_IDENTITY is required" >&2
  exit 1
fi
if [ -z "${APPLE_ID:-}" ]; then
  echo "error: APPLE_ID is required" >&2
  exit 1
fi
if [ -z "${APPLE_TEAM_ID:-}" ]; then
  echo "error: APPLE_TEAM_ID is required" >&2
  exit 1
fi
if [ -z "${APPLE_APP_PASSWORD:-}" ]; then
  echo "error: APPLE_APP_PASSWORD is required (app-specific password)" >&2
  exit 1
fi

codesign --force --options runtime --timestamp --sign "$MACOS_SIGN_IDENTITY" "$BIN"
rm -f "$ZIP"
zip -j "$ZIP" "$BIN"
xcrun notarytool submit "$ZIP" --apple-id "$APPLE_ID" --team-id "$APPLE_TEAM_ID" --password "$APPLE_APP_PASSWORD" --wait
echo "Notarization complete: $ZIP"
