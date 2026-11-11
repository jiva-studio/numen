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
# APT_RETRY_DEADLINE seconds the whole run may take, refreshes and waits included (default 300)
# APT_RETRY_DELAYS   seconds to wait before attempt 2, 3, 4 (default "10 20 40")
# APT_RETRY_CONF     file the apt options are written to
# APT_RETRY_UPDATE   the command that refreshes the package lists
set -u

deadline=${APT_RETRY_DEADLINE:-300}
delays=(${APT_RETRY_DELAYS:-10 20 40})
conf=${APT_RETRY_CONF:-/etc/apt/apt.conf.d/99-numen-network}
read -r -a update <<< "${APT_RETRY_UPDATE:-sudo apt-get update -qq}"

update_first=false
if [ "${1:-}" = --update ]; then
  update_first=true
  shift
fi
if [ $# -eq 0 ]; then
  echo "usage: apt-retry.sh [--update] command [argument...]" >&2
  exit 2
fi

options='Acquire::Retries "1"; Acquire::http::Timeout "30"; Acquire::https::Timeout "30";'
if [ -w "$(dirname "$conf")" ]; then
  echo "$options" > "$conf"
else
  echo "$options" | sudo tee "$conf" > /dev/null
fi

log=$(mktemp)
trap 'rm -f "$log"' EXIT
attempts=$((${#delays[@]} + 1))
status=0
SECONDS=0
timed_out=false
# Runs a command for at most the time left of the deadline.
run_bounded() {
  local left=$((deadline - SECONDS))
  if [ "$left" -le 0 ]; then
    timed_out=true
    return 124
  fi
  timeout -k 5 "$left" "$@"
}
for ((attempt = 1; attempt <= attempts; attempt++)); do
  if [ "$attempt" -gt 1 ]; then
    wait=${delays[$((attempt - 2))]}
    if [ $((SECONDS + wait)) -ge "$deadline" ]; then
      timed_out=true
      break
    fi
    echo "apt attempt $((attempt - 1)) of $attempts failed (exit $status); waiting ${wait}s" >&2
    sleep "$wait"
  fi
  if [ "$attempt" -gt 1 ] || [ "$update_first" = true ]; then
    # A failed refresh is not final: the command that follows fails or succeeds on its own.
    run_bounded "${update[@]}" 2>&1 || true
  fi
  run_bounded "$@" 2>&1 | tee "$log"
  status=${PIPESTATUS[0]}
  if [ "$status" -eq 0 ]; then
    exit 0
  fi
  if [ "$status" -eq 124 ]; then
    timed_out=true
    break
  fi
done

if [ "$timed_out" = true ]; then
  echo "::error::apt did not finish within ${deadline}s (last exit $status): $*"
  echo "Output of the last attempt:"
  cat "$log"
  exit 124
fi
echo "::error::apt failed after $attempts attempts (exit $status): $*"
echo "Output of the last attempt:"
cat "$log"
exit "$status"
