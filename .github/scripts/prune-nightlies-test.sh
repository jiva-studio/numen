#!/usr/bin/env bash
# Runs prune-nightlies.sh against a local fake of Bunny storage.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
server=
trap '[[ -n $server ]] && kill "$server" 2>/dev/null; rm -rf "$work"' EXIT

nightly() { printf '2026.11.0-nightly.%s-%s-abc%04d\n' "$1" "$1" "$1"; }

start() {
  rm -f "$work/port"
  : > "$work/log"
  node "$here/prune-nightlies-fake-bunny.mjs" "$work/log" "$work/port" "$@" &
  server=$!
  for _ in $(seq 100); do [[ -s $work/port ]] && break; sleep 0.1; done
  [[ -s $work/port ]] || { echo "FAIL: the fake server did not start"; exit 1; }
}

stop() { kill "$server"; wait "$server" 2>/dev/null || true; server=; }

run() {
  BUNNY_STORAGE_SCHEME=http BUNNY_STORAGE_ENDPOINT="127.0.0.1:$(cat "$work/port")" \
    BUNNY_STORAGE_ZONE=zone BUNNY_STORAGE_KEY=test-key "$@" "$here/prune-nightlies.sh" >"$work/out" 2>&1
}

fail() { echo "FAIL: $1"; cat "$work/out"; exit 1; }

stable=(2026.10.0-100-aaaaaaa 2026.9.3-90-bbbbbbb checksums.txt)

mapfile -t all < <(for n in $(seq 1 35); do nightly "$n"; done)
start "${all[@]}" "${stable[@]}"
if ! run env; then fail "pruning 35 nightlies failed"; fi
deleted=$(grep -c '^DELETE' "$work/log" || true)
[[ $deleted == 5 ]] || fail "deleted $deleted, want 5"
for n in 1 2 3 4 5; do
  grep -q "^DELETE /zone/builds/$(nightly "$n")/ keyed" "$work/log" || fail "nightly $n was not deleted"
done
grep -q '^GET /zone/builds/ keyed' "$work/log" || fail "the list request carried no key"
if grep '^DELETE' "$work/log" | grep -q -e 'nightly\.\(3[0-5]\|[6-9]\|[12][0-9]\)-' ; then
  grep '^DELETE' "$work/log" | grep -q -e 'nightly\.[1-5]-' && true
fi
for n in 6 35; do
  ! grep -q "^DELETE /zone/builds/$(nightly "$n")/" "$work/log" || fail "nightly $n was deleted"
done
! grep -q 'aaaaaaa\|bbbbbbb\|checksums' <(grep '^DELETE' "$work/log") || fail "a stable entry was deleted"
echo "ok: keeps 30 of 35 nightlies and never deletes stable stamps"
stop

start "${all[@]}" "${stable[@]}"
run env DRY_RUN=true || fail "dry run failed"
! grep -q '^DELETE' "$work/log" || fail "dry run deleted"
grep -q 'would delete builds/' "$work/out" || fail "dry run printed nothing"
echo "ok: dry run deletes nothing"
: > "$work/log"
BUNNY_STORAGE_SCHEME=http BUNNY_STORAGE_ENDPOINT="127.0.0.1:$(cat "$work/port")" BUNNY_STORAGE_ZONE=zone \
  BUNNY_STORAGE_KEY=test-key "$here/prune-nightlies.sh" --dry-run >"$work/out" 2>&1 || fail "--dry-run failed"
! grep -q "^DELETE" "$work/log" || fail "--dry-run deleted"
echo "ok: --dry-run deletes nothing"
stop

start "${all[@]}" "${stable[@]}" 2026.11.0-nightly.x-x-zzz
if run env; then fail "an unparsable stamp did not fail"; fi
! grep -q '^DELETE' "$work/log" || fail "an unparsable stamp deleted something"
echo "ok: fails closed on an unparsable name"
stop

start "${all[@]}" 2026.11.0-nightly.40-41-abc0040
if run env; then fail "a stamp with two different build numbers did not fail"; fi
! grep -q '^DELETE' "$work/log" || fail "a mismatched stamp deleted something"
echo "ok: fails closed on a stamp whose build numbers differ"
stop

start "${all[@]}"
if BUNNY_STORAGE_SCHEME=http BUNNY_STORAGE_ENDPOINT="127.0.0.1:$(cat "$work/port")" BUNNY_STORAGE_ZONE=zone \
  BUNNY_STORAGE_KEY=wrong "$here/prune-nightlies.sh" >"$work/out" 2>&1; then fail "a wrong key did not fail"; fi
! grep -q 'wrong' "$work/out" || fail "the key was printed"
! grep -q '^DELETE' "$work/log" || fail "a wrong key deleted something"
echo "ok: sends the key and never prints it"
stop
