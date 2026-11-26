#!/usr/bin/env bash
# Runs the Go tests of plugins/tools-sync with the given go test arguments, writes a count line to the job summary, and prints the failing output only when a test failed.
# usage: live-go-tests.sh LABEL [go test arguments]
set -euo pipefail

label=$1
shift
out=${RUNNER_TEMP:-/tmp}/go-test-$$.json
summary=${GITHUB_STEP_SUMMARY:-/dev/null}

status=0
(cd plugins/tools-sync && go test -json "$@") >"$out" || status=$?
count() { jq -s --arg action "$1" '[.[] | select(.Action == $action and .Test != null)] | length' "$out"; }
echo "$label: $(count pass) passed, $(count fail) failed, $(count skip) skipped" | tee -a "$summary"
if [ "$status" -ne 0 ]; then
  jq -r 'select(.Action == "output") | .Output' "$out" | grep -vE '^(=== |\s*--- (PASS|SKIP)|PASS$|ok )' || true
fi
rm -f "$out"
exit "$status"
