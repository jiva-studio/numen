#!/usr/bin/env bash
# Usage: stage-plugins.sh <plugins-dir> <destination>
# Copies manifest.json and whichever of bin and dist exist from each plugin that has a bin or a dist
# folder into <destination>/<plugin>, and copies the plugin index plugins-index.json of <plugins-dir>
# into <destination>. Fails when <plugins-dir> has no index.
set -euo pipefail

plugins=$1
destination=$2

for p in "$plugins"/*; do
  if [ -f "$p/manifest.json" ] && { [ -d "$p/bin" ] || [ -d "$p/dist" ]; }; then
    name=$(basename "$p")
    mkdir -p "$destination/$name"
    cp "$p/manifest.json" "$destination/$name/"
    for folder in bin dist; do
      if [ -d "$p/$folder" ]; then cp -r "$p/$folder" "$destination/$name/"; fi
    done
  fi
done

if [ ! -f "$plugins/plugins-index.json" ]; then
  echo "::error::$plugins has no plugins-index.json: run make plugins_index"
  exit 1
fi
mkdir -p "$destination"
cp "$plugins/plugins-index.json" "$destination/"
