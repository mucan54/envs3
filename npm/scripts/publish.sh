#!/bin/sh
set -eu

# Usage: ./npm/scripts/publish.sh <version>
# Example: ./npm/scripts/publish.sh 0.1.0
#
# Prerequisites:
#   1. Run `make build-npm-all` first (or let GoReleaser build)
#   2. You are logged in to npm (`npm login`)

VERSION="${1:?Usage: publish.sh <version>}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
NPM_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Publishing envs3 v${VERSION}"
echo ""

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

# 2. Copy binaries from dist/ or npm/envs3-*/bin/ into platform packages
echo ""
echo "Checking binaries..."
DIST_DIR="${NPM_DIR}/../dist"

copy_binary() {
  go_name="$1"    # e.g. envs3-linux-amd64
  npm_pkg="$2"    # e.g. envs3-linux-x64
  bin_name="$3"   # e.g. envs3 or envs3.exe

  dest="${NPM_DIR}/${npm_pkg}/bin/${bin_name}"

  # Check if binary already exists (from make build-npm-all)
  if [ -f "$dest" ] && [ -x "$dest" ]; then
    size=$(du -h "$dest" | cut -f1)
    echo "  ${npm_pkg}/bin/${bin_name} (${size}) — already built"
    return
  fi

  # Try GoReleaser dist/
  src="${DIST_DIR}/${go_name}"
  if [ -f "$src" ]; then
    cp "$src" "$dest"
    chmod +x "$dest"
    size=$(du -h "$dest" | cut -f1)
    echo "  ${go_name} -> ${npm_pkg}/bin/ (${size})"
    return
  fi

  echo "  WARNING: ${npm_pkg} binary not found — run 'make build-npm-all' first"
}

copy_binary "envs3-linux-amd64"        "envs3-linux-x64"    "envs3"
copy_binary "envs3-linux-arm64"        "envs3-linux-arm64"  "envs3"
copy_binary "envs3-darwin-amd64"       "envs3-darwin-x64"   "envs3"
copy_binary "envs3-darwin-arm64"       "envs3-darwin-arm64" "envs3"
copy_binary "envs3-windows-amd64.exe"  "envs3-win32-x64"   "envs3.exe"

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
