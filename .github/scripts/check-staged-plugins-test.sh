#!/usr/bin/env bash
# Runs check-staged-plugins.sh on an empty directory, a matching pair and a non-matching pair.
set -euo pipefail

check="$(dirname "$0")/check-staged-plugins.sh"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

make_plugin() {
  mkdir -p "$1/$2"
  echo '{}' > "$1/$2/manifest.json"
}

expect() {
  local want=$1 name=$2
  shift 2
  if "$check" "$@" >/dev/null 2>&1; then got=0; else got=1; fi
  if [ "$got" != "$want" ]; then
    echo "FAIL: $name (exit $got, want $want)"
    exit 1
  fi
  echo "ok: $name"
}

mkdir -p "$work/empty-plugins" "$work/empty-staged"
expect 1 "no plugin has a manifest" "$work/empty-plugins" "$work/empty-staged"

make_plugin "$work/plugins" alpha
make_plugin "$work/plugins" beta
make_plugin "$work/same" alpha
make_plugin "$work/same" beta
expect 0 "staged plugins match" "$work/plugins" "$work/same"

make_plugin "$work/short" alpha
expect 1 "a plugin is not staged" "$work/plugins" "$work/short"

mkdir -p "$work/none"
expect 1 "nothing is staged" "$work/plugins" "$work/none"
