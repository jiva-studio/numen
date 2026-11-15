#!/usr/bin/env bash
set -euo pipefail

if ! command -v sshpass >/dev/null; then
  exec nix shell nixpkgs#sshpass --command "$0" "$@"
fi

here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../../../.." && pwd)"
base_image="${BASE_IMAGE:-ghcr.io/cirruslabs/macos-tahoe-base:latest}"
name="${IMAGE_NAME:-numen-macos-26}"
building="$name-building"
cpu="${CPU:-4}"
memory="${MEMORY:-8192}"
tart="${TART:-tart}"
export TART_NO_AUTO_PRUNE=1
export SSHPASS="${GUEST_PASSWORD:-admin}"
guest_user="${GUEST_USER:-admin}"
ssh_options=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -o ConnectTimeout=5 -o ServerAliveInterval=15 -o ServerAliveCountMax=4)

claude_code_version="${CLAUDE_CODE_VERSION:-2.1.325}"
playwright_version="$(sed -n '/"node_modules\/playwright": {/{n;p;}' "$repo/package-lock.json" | sed -E 's/.*"version": "([^"]+)".*/\1/')"
if [ -z "$playwright_version" ]; then
  echo "build.sh: playwright version not found in $repo/package-lock.json" >&2
  exit 1
fi

workdir="$(mktemp -d)"
tart_pid=""

has_vm() {
  "$tart" list --source local --quiet | grep -qx "$1"
}

stop_building_vm() {
  if [ -n "$tart_pid" ] && kill -0 "$tart_pid" 2>/dev/null; then
    "$tart" stop "$building" --timeout 60 || true
    wait "$tart_pid" || true
  fi
  tart_pid=""
}

cleanup() {
  stop_building_vm
  if has_vm "$building"; then
    "$tart" delete "$building"
  fi
  rm -rf "$workdir"
}
trap cleanup EXIT

guest_ssh() {
  sshpass -e ssh "${ssh_options[@]}" "$guest_user@$ip" "$@"
}

if has_vm "$building"; then
  "$tart" delete "$building"
fi

echo "cloning $base_image to $building"
"$tart" clone "$base_image" "$building"
"$tart" set "$building" --cpu "$cpu" --memory "$memory"

echo "booting $building"
"$tart" run --no-graphics --no-audio "$building" >"$workdir/tart.log" 2>&1 &
tart_pid=$!
ip="$("$tart" ip --wait 300 "$building")"

for _ in $(seq 1 60); do
  guest_ssh true 2>/dev/null && break
  sleep 5
done
guest_ssh true

echo "provisioning $building at $ip"
mkdir -p "$workdir/image/numen"
cp "$here/provision.sh" "$here/claude" "$here/numen-runner-bootstrap" "$here/md.numen.runner-bootstrap.plist" "$workdir/image/"
cp "$here/../numen/flake.nix" "$here/../numen/flake.lock" "$workdir/image/numen/"
guest_ssh "rm -rf /tmp/numen-image"
sshpass -e scp -r "${ssh_options[@]}" "$workdir/image" "$guest_user@$ip:/tmp/numen-image"
guest_ssh "PLAYWRIGHT_VERSION=$playwright_version CLAUDE_CODE_VERSION=$claude_code_version bash /tmp/numen-image/provision.sh"
guest_ssh "rm -rf /tmp/numen-image"

echo "shutting down $building"
guest_ssh "sudo shutdown -h now" || true
for _ in $(seq 1 120); do
  kill -0 "$tart_pid" 2>/dev/null || break
  sleep 1
done
stop_building_vm

if has_vm "$name"; then
  "$tart" delete "$name"
fi
"$tart" rename "$building" "$name"
echo "image ready: $name"
