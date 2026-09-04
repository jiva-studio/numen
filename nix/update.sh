#!/usr/bin/env bash
# Point the flake at the newest build on a channel. Run it after a release.
#
#   nix/update.sh            the stable channel
#   nix/update.sh beta       the shelf betas stand on
#
# The shelf says what it holds, and the file it names carries its own hash, so
# nothing is fetched here.
set -euo pipefail

channel=${1:-stable}
case "$channel" in
  stable) shelf=latest ;;
  beta) shelf=beta ;;
  *) echo "there is no $channel channel" >&2; exit 1 ;;
esac

served_from=${NUMEN_DOWNLOADS_BASE:-https://dl.numen.md}
at=$(dirname "$0")/release.json

said=$(curl -fsS "$served_from/$shelf/latest.json")

version=$(printf '%s' "$said" | jq -r .version)
url=$(printf '%s' "$said" | jq -r '.files[] | select(.name == "numen-linux-amd64.tar.gz") | .url')
sum=$(printf '%s' "$said" | jq -r '.files[] | select(.name == "numen-linux-amd64.tar.gz") | .sha256')

if [ -z "$url" ] || [ "$url" = null ]; then
  echo "the $channel shelf holds no Linux build" >&2
  exit 1
fi

hash=$(nix hash convert --hash-algo sha256 --to sri --from base16 "$sum")

jq -n --arg version "$version" --arg url "$url" --arg hash "$hash" \
  '{ $version, $url, $hash }' > "$at"

cat "$at"
