#!/bin/bash
set -euo pipefail

src="$(cd "$(dirname "$0")" && pwd)"
: "${PLAYWRIGHT_VERSION:?PLAYWRIGHT_VERSION is not set}"

if [ ! -x /nix/var/nix/profiles/default/bin/nix ]; then
  curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install --no-confirm
fi
export PATH="$HOME/.nix-profile/bin:/nix/var/nix/profiles/default/bin:$PATH"

nix profile install "path:$src/numen#default"
node --version
git --version

npx --yes "playwright@$PLAYWRIGHT_VERSION" install chromium webkit

sudo mkdir -p /usr/local/bin
sudo install -m 755 "$src/numen-runner-bootstrap" /usr/local/bin/numen-runner-bootstrap
mkdir -p "$HOME/Library/LaunchAgents"
install -m 644 "$src/md.numen.runner-bootstrap.plist" "$HOME/Library/LaunchAgents/md.numen.runner-bootstrap.plist"
