#!/usr/bin/env bash
# Runs an apt command and runs it again when it fails, for the transient network errors of a
# runner. A failure that lasts through every attempt fails the step with the output of the last one.
#
#   apt-retry.sh [--update] command [argument...]
#
# --update runs `apt-get update` before the first attempt; every later attempt is preceded by one.
# The apt options below are written to apt's configuration first, so apt calls made by the command
# itself (npx playwright install-deps) retry and time out the same way.
#
# APT_RETRY_DELAYS   seconds to wait before attempt 2, 3, 4 (default "10 20 40")
# APT_RETRY_CONF     file the apt options are written to
# APT_RETRY_UPDATE   the command that refreshes the package lists
set -u

delays=(${APT_RETRY_DELAYS:-10 20 40})
conf=${APT_RETRY_CONF:-/etc/apt/apt.conf.d/99-numen-network}
update=${APT_RETRY_UPDATE:-sudo apt-get update -qq}

update_first=false
if [ "${1:-}" = --update ]; then
  update_first=true
  shift
fi
if [ $# -eq 0 ]; then
  echo "usage: apt-retry.sh [--update] command [argument...]" >&2
  exit 2
fi

options='Acquire::Retries "5"; Acquire::http::Timeout "30"; Acquire::https::Timeout "30";'
if [ -w "$(dirname "$conf")" ]; then
  echo "$options" > "$conf"
else
  echo "$options" | sudo tee "$conf" > /dev/null
fi

log=$(mktemp)
trap 'rm -f "$log"' EXIT
attempts=$((${#delays[@]} + 1))
status=0
for ((attempt = 1; attempt <= attempts; attempt++)); do
  if [ "$attempt" -gt 1 ]; then
    wait=${delays[$((attempt - 2))]}
    echo "apt attempt $((attempt - 1)) of $attempts failed (exit $status); waiting ${wait}s" >&2
    sleep "$wait"
  fi
  if [ "$attempt" -gt 1 ] || [ "$update_first" = true ]; then
    # A failed refresh is not final: the command that follows fails or succeeds on its own.
    $update 2>&1 || true
  fi
  "$@" 2>&1 | tee "$log"
  status=${PIPESTATUS[0]}
  if [ "$status" -eq 0 ]; then
    exit 0
  fi
done

echo "::error::apt failed after $attempts attempts (exit $status): $*"
echo "Output of the last attempt:"
cat "$log"
exit "$status"
