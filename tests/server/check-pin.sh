#!/usr/bin/env bash
# Fails when a pinned image of stack.lock cannot be read from the registry or the pin is older than 30 days. Does nothing while the digests are empty.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
max_days=30

image=
devtools=
pinned_on=
while IFS='=' read -r key value; do
  case "$key" in
    NUMEN_SYNC_IMAGE) image=$value ;;
    NUMEN_SYNC_DEVTOOLS_IMAGE) devtools=$value ;;
    NUMEN_SYNC_PINNED_ON) pinned_on=$value ;;
  esac
done <"$here/stack.lock"

if [ -z "$image" ] && [ -z "$devtools" ]; then
  echo "stack.lock holds no digests: nothing to check"
  exit 0
fi
if [ -z "$image" ] || [ -z "$devtools" ] || [ -z "$pinned_on" ]; then
  echo "stack.lock is half filled: run make sync_stack_bump" >&2
  exit 1
fi

if ! pinned_epoch=$(date -u -d "$pinned_on" +%s 2>/dev/null || date -u -j -f %Y-%m-%d "$pinned_on" +%s 2>/dev/null); then
  echo "NUMEN_SYNC_PINNED_ON is not a date (YYYY-MM-DD): $pinned_on" >&2
  exit 1
fi
age_days=$((($(date -u +%s) - pinned_epoch) / 86400))
if [ "$age_days" -gt "$max_days" ]; then
  echo "the pin of stack.lock is $age_days days old (limit $max_days): run make sync_stack_bump" >&2
  exit 1
fi

for ref in "$image" "$devtools"; do
  if ! docker buildx imagetools inspect "$ref" >/dev/null 2>&1; then
    echo "cannot read the pinned image $ref: it is gone from the registry, or the login has no read access; run make sync_stack_bump" >&2
    if [ "${GITHUB_ACTIONS:-}" = true ]; then
      echo "::error title=No access to the server images::The pinned image $ref cannot be read. If it exists, add the repository jiva-studio/numen with Read in the package settings of both numen-sync and numen-sync-devtools (Packages, package settings, Manage Actions access); otherwise run make sync_stack_bump."
    fi
    exit 1
  fi
done
echo "stack.lock is $age_days days old and both images are readable"
