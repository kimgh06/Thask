#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:?Usage: ./scripts/release.sh <version>}"
VERSION_NUM="${VERSION#v}"  # strip leading 'v' if present

echo "=== Building Thask CLI ${VERSION} ==="

# Build cross-platform binaries
make release-cli

# Package each binary as .tar.gz
cd dist
for binary in thask-*; do
    if [[ "$binary" == *.exe ]]; then
        tarname="${binary%.exe}.tar.gz"
        tar czf "$tarname" "$binary"
    else
        tarname="${binary}.tar.gz"
        tar czf "$tarname" "$binary"
    fi
    echo "  Packaged: $tarname"
done

echo ""
echo "=== Artifacts in dist/ ==="
ls -lh dist/*.tar.gz 2>/dev/null || ls -lh *.tar.gz

echo ""
echo "=== Next steps ==="
echo "1. Create GitHub release: gh release create v${VERSION_NUM} dist/*.tar.gz --title 'v${VERSION_NUM}' --generate-notes"
echo "2. Update Formula/thask.rb with new version and SHA256 hashes:"
echo "   shasum -a 256 dist/*.tar.gz"
echo "3. Push formula to homebrew-thask tap repo"
