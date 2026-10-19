#!/usr/bin/env bash
# Usage: check-staged-plugins.sh <plugins-dir> <staged-dir>
# Fails unless the plugins in <staged-dir> are exactly the plugins of <plugins-dir> that have a
# manifest.json, and fails when <plugins-dir> has none.
set -euo pipefail
shopt -s nullglob

plugins=$1
staged=$2

list_plugins() {
  local manifest
  for manifest in "$1"/*/manifest.json; do
    basename "$(dirname "$manifest")"
  done | sort
}

declared=$(list_plugins "$plugins")
shipped=$(list_plugins "$staged")

if [ -z "$declared" ]; then
  echo "::error::no plugin with a manifest.json in $plugins"
  exit 1
fi

if [ "$declared" != "$shipped" ]; then
  echo "::error::the staged plugins differ from the plugins with a manifest"
  diff <(echo "$declared") <(echo "$shipped") || true
  exit 1
fi
echo "$(echo "$declared" | wc -l) plugins staged"
