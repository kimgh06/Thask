#!/usr/bin/env bash
# Sync plugin.json version with the canonical CLI version (git tag).
# Usage: ./scripts/sync-version.sh [version]
#   With no arg, derives from `git describe --tags`.

set -euo pipefail

VERSION="${1:-$(git describe --tags --always 2>/dev/null || echo "0.0.0-dev")}"
VERSION="${VERSION#v}"

PLUGIN_JSON="$(dirname "$0")/../.claude-plugin/plugin.json"

if [ ! -f "$PLUGIN_JSON" ]; then
  echo "plugin.json not found at $PLUGIN_JSON" >&2
  exit 1
fi

# Portable in-place edit (works on macOS/Linux without GNU sed quirks).
tmp="$(mktemp)"
awk -v v="$VERSION" '
  /"version":/ && !done { sub(/"version": *"[^"]*"/, "\"version\": \"" v "\""); done=1 }
  { print }
' "$PLUGIN_JSON" > "$tmp"
mv "$tmp" "$PLUGIN_JSON"

echo "plugin.json version → $VERSION"
