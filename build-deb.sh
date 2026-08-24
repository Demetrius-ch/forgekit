#!/bin/bash
# Build script for ForgeKit Debian package
# Run this in a terminal where sudo works

set -euo pipefail

echo "=== Installing build dependencies ==="
sudo apt-get update
sudo apt-get install -y debhelper dh-golang golang-go git

echo "=== Building package ==="
cd "$(dirname "$0")"
dpkg-buildpackage -us -uc -b

echo "=== Running lintian ==="
lintian ../forgekit_*_amd64.changes

echo "=== Package contents ==="
dpkg-deb -c ../forge_*_amd64.deb

echo "=== Package info ==="
dpkg-deb -I ../forge_*_amd64.deb

echo "=== Build complete ==="
echo "Package: ../forge_*_amd64.deb"
echo "Changes: ../forgekit_*_amd64.changes"