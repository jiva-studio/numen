#!/usr/bin/env bash
# Runs a real acceptance group and prints the counts from its JSON report.
set -euo pipefail

group=$1
settings=$(make -C "$GITHUB_WORKSPACE" -s sync_stack_env PROJECT="${NUMEN_SYNC_OWN_STACK:-}")
eval "$settings"

status=0
read -ra features <<< "$FEATURES"
args=()
if [ "$group" = acceptance:stand ]; then args=(--retries=0); fi
npm run "$group" -- "${args[@]}" --project=chromium --max-failures=0 --reporter=list,json,blob "${features[@]}" || status=$?
if [ -s "$PLAYWRIGHT_JSON_OUTPUT_NAME" ]; then
  counts=$(jq -r '.stats | "\(.expected) passed, \(.unexpected) failed, \(.flaky) flaky, \(.skipped) skipped"' "$PLAYWRIGHT_JSON_OUTPUT_NAME")
  echo "$group: $counts" | tee -a "$GITHUB_STEP_SUMMARY"
fi
exit "$status"
