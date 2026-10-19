#!/usr/bin/env bash
# Usage: check-staged-plugins.sh <plugins-dir> <staged-dir>
# Fails unless the plugins in <staged-dir> are exactly the plugins of <plugins-dir> that have a
# manifest.json.
set -euo pipefail

plugins=$1
staged=$2

declared=$(for m in "$plugins"/*/manifest.json; do basename "$(dirname "$m")"; done | sort)
shipped=$(for m in "$staged"/*/manifest.json; do [ -f "$m" ] && basename "$(dirname "$m")"; done | sort || true)

if [ "$declared" != "$shipped" ]; then
  echo "::error::the staged plugins differ from the plugins with a manifest"
  diff <(echo "$declared") <(echo "$shipped") || true
  exit 1
fi
echo "$(echo "$declared" | wc -l) plugins staged"
