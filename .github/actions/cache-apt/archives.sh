#!/usr/bin/env bash
# Installs the .deb files cache-apt restored, and keeps the folder to the versions installed.
#
#   archives.sh install   installs every restored .deb file in one apt transaction, without
#                         downloading; fails when they do not install together over the packages
#                         of the runner, which apt finds before dpkg changes anything
#   archives.sh prune     removes the .deb files whose version is not installed, so the cache a
#                         run saves holds the one set of versions that installed together
#
# A cache restored by a prefix comes from another runner image, and the mirrors may have moved on
# since it was saved, so its files can be newer, older or a mix. `dpkg -i` unpacks such a set in
# file order and stops at a Breaks between versions, which leaves dpkg with unconfigured packages
# that every later apt call refuses to work around.
#
# APT_ARCHIVES the folder of the .deb files (default ~/.cache/apt-archives)
set -euo pipefail
shopt -s nullglob

dir=${APT_ARCHIVES:-$HOME/.cache/apt-archives}
debs=("$dir"/*.deb)

case "${1:-}" in
  install)
    if [ ${#debs[@]} -eq 0 ]; then
      exit 1
    fi
    if sudo apt-get install -y --no-download --no-install-recommends \
      -o Dpkg::Options::=--force-unsafe-io "${debs[@]}"; then
      exit 0
    fi
    # A maintainer script that failed leaves packages unpacked but not configured.
    sudo dpkg --configure -a || true
    if ! sudo apt-get check > /dev/null 2>&1; then
      "$(dirname "$0")/../../scripts/apt-retry.sh" --update sudo apt-get install -f -y --no-install-recommends
    fi
    exit 1
    ;;
  prune)
    for deb in "${debs[@]}"; do
      read -r name version < <(dpkg-deb --show --showformat='${Package} ${Version}\n' "$deb")
      if ! dpkg-query -W -f='${db:Status-Abbrev}${Version}\n' "$name" 2> /dev/null | grep -qxF "ii $version"; then
        rm -f "$deb"
      fi
    done
    ;;
  *)
    echo "usage: archives.sh install|prune" >&2
    exit 2
    ;;
esac
