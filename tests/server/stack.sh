#!/usr/bin/env bash
# Brings the test stack of the real sync server up and down, and prints the settings the tests read.
# usage: stack.sh up | down | env
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
project=numen-sync-test
url=http://127.0.0.1:8787
payment_url=http://127.0.0.1:8789
bucket_url=http://127.0.0.1:8790

read_lock() {
  local line key
  while IFS= read -r line; do
    case "$line" in '' | '#'*) continue ;; esac
    key=${line%%=*}
    if [ -z "${!key:-}" ]; then export "$key=${line#*=}"; fi
  done <"$here/stack.lock"
  if [ -z "${NUMEN_SYNC_IMAGE:-}" ] || [ -z "${NUMEN_SYNC_DEVTOOLS_IMAGE:-}" ]; then
    echo "no server images: run make sync_stack_bump, or set NUMEN_SYNC_IMAGE and NUMEN_SYNC_DEVTOOLS_IMAGE" >&2
    exit 1
  fi
}

compose() { docker compose -f "$here/compose.yaml" "$@"; }

case "${1:-}" in
  up)
    read_lock
    compose up --detach --wait --quiet-pull
    ;;
  down)
    read_lock
    compose down --volumes
    ;;
  env)
    read_lock
    root=$(docker run --rm -v "${project}_keys:/keys:ro" --entrypoint cat "$NUMEN_SYNC_DEVTOOLS_IMAGE" /keys/root-public.hex)
    echo "export NUMEN_SYNC_TEST_URL=$url"
    echo "export NUMEN_SYNC_TEST_ROOT_KEY=$root"
    echo "export NUMEN_SYNC_TEST_PAYMENT_URL=$payment_url"
    echo "export NUMEN_SYNC_TEST_BUCKET_URL=$bucket_url"
    ;;
  *)
    echo "usage: stack.sh up | down | env" >&2
    exit 2
    ;;
esac
