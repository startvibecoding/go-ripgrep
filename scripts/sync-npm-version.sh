#!/bin/bash

# Sync version from git tag to npm package.json (main + all platform packages)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
NPM_DIR="$PROJECT_ROOT/npm"
PACKAGE_JSON="$NPM_DIR/package.json"

# Get version from argument or git
if [ -n "$1" ]; then
  VERSION="$1"
else
  VERSION=$(git describe --tags --always 2>/dev/null)
  if [ -z "$VERSION" ]; then
    VERSION="0.0.1"
  fi
fi
VERSION="${VERSION#v}"
VERSION="${VERSION%-dirty}"
VERSION="${VERSION%%-[0-9]*-g[0-9a-f]*}"

echo "Syncing npm version to: $VERSION"

# Update main package.json version + optionalDependencies
node -e "
const fs = require('fs');
const pkgPath = '$PACKAGE_JSON';
if (fs.existsSync(pkgPath)) {
  const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
  pkg.version = '$VERSION';
  if (pkg.optionalDependencies) {
    for (const key of Object.keys(pkg.optionalDependencies)) {
      pkg.optionalDependencies[key] = '$VERSION';
    }
  }
  fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');
  console.log('Updated: $PACKAGE_JSON');
} else {
  console.log('Main package.json not found, skipping sync');
}
"

# Update all platform package.json files
PACKAGES_DIR="$NPM_DIR/packages"
if [ -d "$PACKAGES_DIR" ]; then
  while IFS= read -r -d '' pkgPath; do
    node -e "
    const fs = require('fs');
    const pkgPath = '$pkgPath';
    const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
    pkg.version = '$VERSION';
    fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');
    console.log('Updated: ' + pkgPath);
    "
  done < <(find "$PACKAGES_DIR" -mindepth 2 -maxdepth 4 -name package.json -print0)
fi

echo "Version sync complete: $VERSION"
