#!/bin/bash
set -euo pipefail

# Publish all npm packages
# Usage: ./npm/scripts/publish.sh [--dry-run]

NPM_DIR="npm"
DRY_RUN=""
if [[ "${1:-}" == "--dry-run" ]]; then
  DRY_RUN="--dry-run"
  echo "DRY RUN mode"
fi

# Publish platform packages first
for PKG_DIR in "${NPM_DIR}"/cli-*; do
  if [ -d "$PKG_DIR" ] && [ -f "$PKG_DIR/package.json" ]; then
    PKG_NAME=$(node -p "require('./${PKG_DIR}/package.json').name")
    PKG_VER=$(node -p "require('./${PKG_DIR}/package.json').version")
    echo "Publishing ${PKG_NAME}@${PKG_VER}"
    cd "$PKG_DIR"
    npm publish --access public $DRY_RUN || echo "  (already published or error)"
    cd - > /dev/null
  fi
done

# Publish main package last
echo "Publishing thask@$(node -p "require('./${NPM_DIR}/thask/package.json').version")"
cd "${NPM_DIR}/thask"
npm publish --access public $DRY_RUN || echo "  (already published or error)"
cd - > /dev/null

echo ""
echo "Done."
