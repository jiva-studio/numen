#!/usr/bin/env bash
# Keeps the newest nightly builds under builds/ on Bunny storage and deletes the older ones.
# Usage: prune-nightlies.sh [--dry-run]
# Environment: BUNNY_STORAGE_ENDPOINT, BUNNY_STORAGE_ZONE, BUNNY_STORAGE_KEY, KEEP (default 30),
# DRY_RUN (true prints only), BUNNY_STORAGE_SCHEME (default https).
set -euo pipefail
set +x

keep=${KEEP:-30}
dry=${DRY_RUN:-false}
[[ ${1:-} == --dry-run ]] && dry=true
scheme=${BUNNY_STORAGE_SCHEME:-https}
base="$scheme://${BUNNY_STORAGE_ENDPOINT:?}/${BUNNY_STORAGE_ZONE:?}/builds"
: "${BUNNY_STORAGE_KEY:?}"
[[ $keep =~ ^[1-9][0-9]*$ ]] || { echo "::error::KEEP must be a positive number"; exit 1; }

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
umask 077
printf 'AccessKey: %s\n' "$BUNNY_STORAGE_KEY" > "$work/header"

code=$(curl -sS -o "$work/list.json" -w '%{http_code}' -H @"$work/header" "$base/")
if [[ $code != 200 ]]; then
  echo "::error::listing builds/ was refused with $code"
  exit 1
fi

jq -r '.[] | select(.IsDirectory) | .ObjectName' "$work/list.json" > "$work/names"

pattern='^[0-9]+\.[0-9]+\.[0-9]+-nightly\.([0-9]+)-([0-9]+)-[0-9a-f]+$'
: > "$work/nightlies"
while IFS= read -r name; do
  name=${name%/}
  [[ $name == *-nightly.* ]] || continue
  if [[ ! $name =~ $pattern || ${BASH_REMATCH[1]} != "${BASH_REMATCH[2]}" ]]; then
    echo "::error::cannot parse the stamp '$name'; nothing is deleted"
    exit 1
  fi
  printf '%s %s\n' "${BASH_REMATCH[2]}" "$name" >> "$work/nightlies"
done < "$work/names"

total=$(wc -l < "$work/nightlies")
echo "$total nightly builds on the CDN, keeping $keep"

sort -k1,1nr "$work/nightlies" | tail -n +"$((keep + 1))" | cut -d' ' -f2 > "$work/doomed"

failed=0
while IFS= read -r name; do
  if [[ $dry == true ]]; then
    echo "would delete builds/$name"
    continue
  fi
  echo "delete builds/$name"
  code=$(curl -sS -o /dev/null -w '%{http_code}' -X DELETE -H @"$work/header" "$base/$name/")
  case "$code" in
    20*) ;;
    *) echo "::error::deleting builds/$name was refused with $code"; failed=1 ;;
  esac
done < "$work/doomed"

exit "$failed"
