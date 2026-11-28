#!/usr/bin/env bash
# Prints the hash of what the end-to-end build is made from: the blobs of the files
# `select-tests.mjs inputs` names, less the scenarios and the tests (the binaries and the plugins do
# not contain them), plus the Markdown files and the base tsconfig the sources read, and the actions
# and the apt script the job runs. Run it in the root of a checkout.
set -euo pipefail

blobs=$(mktemp)
trap 'rm -f "$blobs"' EXIT
git ls-files -s > "$blobs"

{
  cut -f2 "$blobs" | node tests/e2e/desktop/scripts/select-tests.mjs inputs | grep -Ev '^(tests|features)/' || true
  cut -f2 "$blobs" | grep -E '^((apps|libs|plugins|tools)/.*\.mdx?|tools/syncaccount/.*|tsconfig\.base\.json|\.github/actions/.*|\.github/scripts/(apt-retry|hash-e2e-build|run-e2e-build)\.sh)$' || true
} | sort -u | awk -F'\t' 'NR == FNR { keep[$0]; next } $2 in keep' - "$blobs" | sha256sum | cut -d' ' -f1
