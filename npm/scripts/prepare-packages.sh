#!/bin/bash
set -euo pipefail

VERSION="${1:-$(git describe --tags --always 2>/dev/null || echo "0.0.0-dev")}"
VERSION="${VERSION#v}"

DIST_DIR="dist"
NPM_DIR="npm"

echo "Preparing npm packages v${VERSION}"

# go_key:npm_key:os:cpu
TARGETS="
darwin-arm64:darwin-arm64:darwin:arm64
darwin-amd64:darwin-x64:darwin:x64
linux-amd64:linux-x64:linux:x64
linux-arm64:linux-arm64:linux:arm64
windows-amd64:win32-x64:win32:x64
"

for entry in $TARGETS; do
  IFS=':' read -r GO_KEY NPM_KEY OS CPU <<< "$entry"
  PKG_NAME="@thask-org/cli-${NPM_KEY}"
  PKG_DIR="${NPM_DIR}/cli-${NPM_KEY}"

  if [[ "$GO_KEY" == windows-* ]]; then
    SRC_BIN="${DIST_DIR}/thask-${GO_KEY}.exe"
    DST_BIN="bin/thask.exe"
  else
    SRC_BIN="${DIST_DIR}/thask-${GO_KEY}"
    DST_BIN="bin/thask"
  fi

  if [ ! -f "$SRC_BIN" ]; then
    echo "SKIP: $SRC_BIN not found"
    continue
  fi

  echo "Creating ${PKG_NAME} from ${SRC_BIN}"

  mkdir -p "${PKG_DIR}/bin"
  cp "$SRC_BIN" "${PKG_DIR}/${DST_BIN}"
  chmod +x "${PKG_DIR}/${DST_BIN}"

  cat > "${PKG_DIR}/package.json" <<EOF
{
  "name": "${PKG_NAME}",
  "version": "${VERSION}",
  "description": "Thask CLI binary for ${OS} ${CPU}",
  "license": "MIT",
  "repository": {
    "type": "git",
    "url": "https://github.com/kimgh06/Thask"
  },
  "os": ["${OS}"],
  "cpu": ["${CPU}"]
}
EOF

  echo "  -> ${PKG_DIR} (${OS}/${CPU})"
done

# Update main package version + optionalDependencies
sed -i '' "s/\"version\": \".*\"/\"version\": \"${VERSION}\"/" "${NPM_DIR}/thask/package.json"
for entry in $TARGETS; do
  IFS=':' read -r _ NPM_KEY _ _ <<< "$entry"
  sed -i '' "s|\"@thask-org/cli-${NPM_KEY}\": \".*\"|\"@thask-org/cli-${NPM_KEY}\": \"${VERSION}\"|" "${NPM_DIR}/thask/package.json"
done

echo ""
echo "Done. Packages ready in ${NPM_DIR}/"
echo "Publish with: ./npm/scripts/publish.sh"
