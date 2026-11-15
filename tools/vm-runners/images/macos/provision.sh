#!/bin/bash
set -euo pipefail

src="$(cd "$(dirname "$0")" && pwd)"
: "${PLAYWRIGHT_VERSION:?PLAYWRIGHT_VERSION is not set}"
: "${RUNNER_VERSION:?RUNNER_VERSION is not set}"
: "${RUNNER_SHA256:?RUNNER_SHA256 is not set}"

home="$HOME"
job_path="$home/.nix-profile/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin"

log() {
  printf '%s provision.sh: %s\n' "$(date '+%H:%M:%S')" "$*"
}

if [ ! -x /nix/var/nix/profiles/default/bin/nix ]; then
  log "installing nix"
  curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install --no-confirm
fi
export PATH="$home/.nix-profile/bin:/nix/var/nix/profiles/default/bin:$PATH"

log "installing the nix profile"
nix profile install "path:$src/numen#default"
node --version
git --version

log "installing playwright browsers"
npx --yes "playwright@$PLAYWRIGHT_VERSION" install chromium webkit

log "installing the actions runner $RUNNER_VERSION"
rm -rf "$home/actions-runner"
mkdir -p "$home/actions-runner"
archive="$home/actions-runner/actions-runner-osx-arm64-$RUNNER_VERSION.tar.gz"
curl --proto '=https' --tlsv1.2 -fsSL -o "$archive" "https://github.com/actions/runner/releases/download/v$RUNNER_VERSION/actions-runner-osx-arm64-$RUNNER_VERSION.tar.gz"
printf '%s  %s\n' "$RUNNER_SHA256" "$archive" | shasum -a 256 -c -
tar -C "$home/actions-runner" -xzf "$archive"
rm -f "$archive"
printf 'PATH=%s\n' "$job_path" >"$home/actions-runner/.env"
test -x "$home/actions-runner/run.sh"
log "done"
