#!/usr/bin/env bash
# Full release pipeline: build → tarball → formula → npm → GitHub Release
# Usage: ./scripts/release.sh <version>
#        THASKOTP=<otp> ./scripts/release.sh <version>
set -euo pipefail

VERSION="${1:-$(node -p "require('./cli/package.json').version" 2>/dev/null)}"
if [[ -z "$VERSION" ]]; then
  echo "Usage: ./scripts/release.sh <version>  (e.g. v0.5.4)"
  exit 1
fi
[[ "$VERSION" == v* ]] || VERSION="v${VERSION}"
VERSION_NUM="${VERSION#v}"

echo "=== Building Thask CLI ${VERSION} ==="

# ── 1. Cross-platform build (parallel) ──────────────────────────────────────
mkdir -p dist
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)
CLI_LDFLAGS="-X github.com/thask/cli/internal/cmd.Version=${VERSION_NUM} \
             -X github.com/thask/cli/internal/cmd.Commit=${COMMIT} \
             -X github.com/thask/cli/internal/mcp.Version=${VERSION_NUM}"

build_one() {
  local os="$1" arch="$2" out="$3"
  echo "  Building ${os}/${arch} → dist/${out}"
  (cd cli && GOOS="$os" GOARCH="$arch" go build -ldflags "$CLI_LDFLAGS" -o "../dist/${out}" ./cmd/thask)
  echo "  Done: dist/${out}"
}

build_one darwin  arm64 thask-darwin-arm64     &
build_one darwin  amd64 thask-darwin-amd64     &
build_one linux   amd64 thask-linux-amd64      &
build_one linux   arm64 thask-linux-arm64      &
build_one windows amd64 thask-windows-amd64.exe &
wait
echo "  All builds complete."

# ── 2. Tarballs for Homebrew ─────────────────────────────────────────────────
echo ""
echo "=== Creating tarballs ==="
for name in thask-darwin-arm64 thask-darwin-amd64 thask-linux-arm64 thask-linux-amd64; do
  (cd dist && cp "$name" thask && tar czf "${name}.tar.gz" thask && rm thask)
  echo "  ${name}.tar.gz"
done

SHA_DA=$(shasum -a 256 dist/thask-darwin-arm64.tar.gz | awk '{print $1}')
SHA_DI=$(shasum -a 256 dist/thask-darwin-amd64.tar.gz | awk '{print $1}')
SHA_LA=$(shasum -a 256 dist/thask-linux-arm64.tar.gz  | awk '{print $1}')
SHA_LI=$(shasum -a 256 dist/thask-linux-amd64.tar.gz  | awk '{print $1}')

echo "  darwin-arm64: ${SHA_DA}"
echo "  darwin-amd64: ${SHA_DI}"
echo "  linux-arm64:  ${SHA_LA}"
echo "  linux-amd64:  ${SHA_LI}"

# ── 3. Update Homebrew formula ────────────────────────────────────────────────
echo ""
echo "=== Updating Formula/thask.rb ==="
python3 - "${VERSION_NUM}" "${SHA_DA}" "${SHA_DI}" "${SHA_LA}" "${SHA_LI}" << 'PYEOF'
import sys, re

version, sha_da, sha_di, sha_la, sha_li = sys.argv[1:]
shas = iter([sha_da, sha_di, sha_la, sha_li])

with open('Formula/thask.rb') as f:
    content = f.read()

content = re.sub(r'version "[^"]*"', f'version "{version}"', content)
content = re.sub(r'sha256 "[^"]*"', lambda m: f'sha256 "{next(shas)}"', content)

with open('Formula/thask.rb', 'w') as f:
    f.write(content)
print(f'  Updated Formula/thask.rb to v{version}')
PYEOF

# ── 3b. Sync plugin.json + marketing/server.json so releases stop drifting.
#       Both were stale for 9+ releases (plugin.json frozen at v0.5.7,
#       marketing/server.json lagged the npm package). Bump in-place; the
#       Homebrew commit below sweeps them up alongside Formula/thask.rb.
echo ""
echo "=== Syncing plugin.json + marketing/server.json to ${VERSION_NUM} ==="
( cd plugin/thask-claudecode && ./scripts/sync-version.sh "${VERSION_NUM}" )
python3 - "${VERSION_NUM}" << 'PYEOF'
import sys, re
version = sys.argv[1]
path = 'marketing/server.json'
with open(path) as f:
    content = f.read()
# Both occurrences (top-level + package) follow the same `"version": "X.Y.Z"` shape.
content = re.sub(r'"version": "[^"]*"', f'"version": "{version}"', content)
with open(path, 'w') as f:
    f.write(content)
print(f'  Updated {path} to {version}')
PYEOF

git add Formula/thask.rb plugin/thask-claudecode/.claude-plugin/plugin.json marketing/server.json
git commit -m "chore: sync Homebrew formula + plugin.json + server.json to ${VERSION}"
if git rev-parse "${VERSION}" >/dev/null 2>&1; then
  echo "  Tag ${VERSION} already exists, skipping tag creation"
else
  git tag "${VERSION}"
  echo "  Tagged ${VERSION}"
fi
git push origin main
git push origin "${VERSION}"
echo "  Pushed main + ${VERSION} to origin"

# ── 4. npm publish ────────────────────────────────────────────────────────────
# Uses ~/.npmrc auth token. If the token has "Bypass 2FA for publishing",
# no OTP is needed. Otherwise, set THASKOTP=<6-digit-code>.
echo ""
echo "=== Publishing to npm ==="
if [[ -n "${THASKOTP:-}" ]]; then
  (cd cli && npm publish --access public --otp="${THASKOTP}")
else
  (cd cli && npm publish --access public)
fi

# ── 5. GitHub Release ─────────────────────────────────────────────────────────
echo ""
echo "=== Creating GitHub Release ${VERSION} ==="
if gh release view "${VERSION}" >/dev/null 2>&1; then
  echo "  Release ${VERSION} already exists, skipping"
else
  gh release create "${VERSION}" \
    dist/thask-darwin-arm64.tar.gz \
    dist/thask-darwin-amd64.tar.gz \
    dist/thask-linux-arm64.tar.gz  \
    dist/thask-linux-amd64.tar.gz  \
    dist/thask-windows-amd64.exe   \
    --title "${VERSION}" --generate-notes
fi

echo ""
echo "=== Done! ${VERSION} released ==="
echo ""
echo "Install via:"
echo "  brew tap kimgh06/thask https://github.com/kimgh06/Thask"
echo "  brew install kimgh06/thask/thask"
echo "  npm install -g @thask-org/cli"
ls -lh dist/