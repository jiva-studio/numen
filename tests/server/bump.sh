#!/usr/bin/env bash
# Writes stack.lock: the digest of the tag main of both private packages and today's date. Needs `gh auth login` with the read:packages scope.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
owner=jiva-studio

if ! gh auth status >/dev/null 2>&1; then
  echo "not logged in to GitHub: run gh auth login (scope read:packages)" >&2
  exit 1
fi

read_digest() {
  local package=$1 out digest
  if ! out=$(gh api --paginate "/orgs/$owner/packages/container/$package/versions" --jq '.[] | select(.metadata.container.tags | index("main")) | .name' 2>&1); then
    echo "cannot read package $owner/$package (absent, or the login lacks read:packages): $out" >&2
    exit 1
  fi
  digest=$(printf '%s\n' "$out" | head -n 1)
  case "$digest" in
    sha256:*) printf '%s' "$digest" ;;
    *)
      echo "package $owner/$package has no version tagged main" >&2
      exit 1
      ;;
  esac
}

image=$(read_digest numen-sync)
devtools=$(read_digest numen-sync-devtools)

cat >"$here/stack.lock" <<LOCK
# The images of the real sync server, by digest. they are published as private packages (tag main); \`make sync_stack_bump\` writes this file.
# While the digests are empty, \`make sync_stack_up\` takes the images from NUMEN_SYNC_IMAGE and NUMEN_SYNC_DEVTOOLS_IMAGE in the environment.
NUMEN_SYNC_IMAGE=ghcr.io/$owner/numen-sync@$image
NUMEN_SYNC_DEVTOOLS_IMAGE=ghcr.io/$owner/numen-sync-devtools@$devtools
NUMEN_SYNC_PINNED_ON=$(date -u +%Y-%m-%d)
LOCK
echo "pinned $image and $devtools"
