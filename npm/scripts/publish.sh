#!/bin/bash
set -euo pipefail

# Usage: ./npm/scripts/publish.sh <version>
# Example: ./npm/scripts/publish.sh 0.1.0
#
# Prerequisites:
#   1. GoReleaser has built the binaries (or run `make build-npm-all`)
#   2. You are logged in to npm (`npm login`)

VERSION="${1:?Usage: publish.sh <version>}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
NPM_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Publishing envs3 v${VERSION}"
echo ""

# GoReleaser binary naming -> npm package directory mapping
declare -A BINARY_MAP=(
  ["envs3-linux-amd64"]="envs3-linux-x64"
  ["envs3-linux-arm64"]="envs3-linux-arm64"
  ["envs3-darwin-amd64"]="envs3-darwin-x64"
  ["envs3-darwin-arm64"]="envs3-darwin-arm64"
  ["envs3-windows-amd64.exe"]="envs3-win32-x64"
)

# 1. Update versions in all package.json files
echo "Updating versions..."
for pkg_dir in "$NPM_DIR"/envs3 "$NPM_DIR"/envs3-*/; do
  if [ -f "$pkg_dir/package.json" ]; then
    node -e "
      const fs = require('fs');
      const path = '${pkg_dir}/package.json';
      const pkg = JSON.parse(fs.readFileSync(path, 'utf8'));
      pkg.version = '${VERSION}';
      if (pkg.optionalDependencies) {
        for (const key of Object.keys(pkg.optionalDependencies)) {
          pkg.optionalDependencies[key] = '${VERSION}';
        }
      }
      fs.writeFileSync(path, JSON.stringify(pkg, null, 2) + '\n');
    "
    echo "  $(basename "$pkg_dir") -> ${VERSION}"
  fi
done

# 2. Copy binaries from GoReleaser dist/ into platform packages
echo ""
echo "Copying binaries..."
DIST_DIR="${NPM_DIR}/../dist"

for binary_name in "${!BINARY_MAP[@]}"; do
  npm_pkg="${BINARY_MAP[$binary_name]}"
  src="${DIST_DIR}/${binary_name}"

  if [[ "$binary_name" == *.exe ]]; then
    dest="${NPM_DIR}/${npm_pkg}/bin/envs3.exe"
  else
    dest="${NPM_DIR}/${npm_pkg}/bin/envs3"
  fi

  if [ -f "$src" ]; then
    cp "$src" "$dest"
    chmod +x "$dest"
    size=$(du -h "$dest" | cut -f1)
    echo "  ${binary_name} -> ${npm_pkg}/bin/ (${size})"
  else
    echo "  WARNING: ${src} not found — skipping ${npm_pkg}"
  fi
done

# 3. Publish platform packages first
echo ""
echo "Publishing platform packages..."
for pkg_dir in "$NPM_DIR"/envs3-*/; do
  if [ -f "$pkg_dir/package.json" ]; then
    pkg_name=$(node -e "console.log(require('${pkg_dir}/package.json').name)")
    echo "  Publishing ${pkg_name}..."
    (cd "$pkg_dir" && npm publish --access public)
  fi
done

# 4. Publish main wrapper package last
echo ""
echo "Publishing @mucan54/envs3..."
(cd "$NPM_DIR/envs3" && npm publish --access public)

echo ""
echo "Done! Published envs3 v${VERSION}"
echo ""
echo "Install with:"
echo "  npm install -g @mucan54/envs3"
echo "  npx @mucan54/envs3 --version"
