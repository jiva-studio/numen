#!/usr/bin/env bash
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../../../.." && pwd)"
base_image="${BASE_IMAGE:-ghcr.io/cirruslabs/macos-tahoe-base:latest}"
name="${IMAGE_NAME:-numen-macos-26}"
building="$name-building"
cpu="${CPU:-4}"
memory="${MEMORY:-8192}"
tart="${TART:-tart}"
export TART_NO_AUTO_PRUNE=1

log() {
  printf '%s build.sh: %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

playwright_version="$(sed -n '/"node_modules\/playwright": {/{n;p;}' "$repo/package-lock.json" | sed -E 's/.*"version": "([^"]+)".*/\1/')"
if [ -z "$playwright_version" ]; then
  log "playwright version not found in $repo/package-lock.json" >&2
  exit 1
fi

runner_version="${RUNNER_VERSION:-}"
runner_sha256="${RUNNER_SHA256:-}"
if [ -z "$runner_version" ] || [ -z "$runner_sha256" ]; then
  release="$(curl --proto '=https' --tlsv1.2 -fsSL https://api.github.com/repos/actions/runner/releases/latest)"
  runner_version="$(printf '%s' "$release" | grep -o '"tag_name": *"v[0-9.]*"' | head -n 1 | sed -E 's/.*"v([0-9.]+)"/\1/')"
  runner_sha256="$(printf '%s' "$release" | grep -o 'BEGIN SHA osx-arm64 -->[0-9a-f]\{64\}' | head -n 1 | sed -E 's/.*-->//')"
fi
if [ -z "$runner_version" ] || [ -z "$runner_sha256" ]; then
  log "runner version or sha256 not found in the latest actions/runner release" >&2
  exit 1
fi

workdir="$(mktemp -d)"
tart_pid=""

has_vm() {
  "$tart" list --source local --quiet | grep -qx "$1"
}

is_vm_running() {
  "$tart" list --source local | awk -v n="$1" '$2 == n { print $NF }' | grep -qx running
}

wait_for_stop() {
  for _ in $(seq 1 180); do
    is_vm_running "$building" || return 0
    sleep 2
  done
  return 1
}

cleanup() {
  status=$?
  if [ -n "$tart_pid" ] && kill -0 "$tart_pid" 2>/dev/null; then
    "$tart" stop "$building" --timeout 30 || true
    wait "$tart_pid" || true
  fi
  if [ "$status" -ne 0 ] && has_vm "$building"; then
    log "build failed, deleting $building"
    "$tart" delete "$building"
  fi
  rm -rf "$workdir"
}
trap cleanup EXIT

guest() {
  "$tart" exec "$building" "$@"
}

if has_vm "$building"; then
  log "deleting stale $building"
  "$tart" delete "$building"
fi

log "cloning $base_image to $building"
"$tart" clone "$base_image" "$building"
"$tart" set "$building" --cpu "$cpu" --memory "$memory"

log "booting $building"
"$tart" run --no-graphics --no-audio "$building" >"$workdir/tart.log" 2>&1 &
tart_pid=$!

for _ in $(seq 1 90); do
  guest true 2>/dev/null && break
  sleep 5
done
guest true

log "copying provisioning files"
guest rm -rf /tmp/numen-image
guest mkdir -p /tmp/numen-image/numen
"$tart" exec -i "$building" tar -C /tmp/numen-image -xf - < <(
  tar -C "$here/.." -cf - macos/provision.sh numen/flake.nix numen/flake.lock
)
guest mv /tmp/numen-image/macos/provision.sh /tmp/numen-image/provision.sh

log "provisioning: playwright $playwright_version, actions runner $runner_version"
guest env \
  PLAYWRIGHT_VERSION="$playwright_version" \
  RUNNER_VERSION="$runner_version" \
  RUNNER_SHA256="$runner_sha256" \
  bash /tmp/numen-image/provision.sh
guest rm -rf /tmp/numen-image

log "shutting down $building"
guest sh -c 'sync; sudo shutdown -h now' || true
wait_for_stop
wait "$tart_pid" || true
tart_pid=""

if has_vm "$name"; then
  log "replacing $name"
  "$tart" delete "$name"
fi
"$tart" rename "$building" "$name"
log "image ready: $name"
