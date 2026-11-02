#!/usr/bin/env bash
# Records the screen of a real WebKitGTK window while the EPUB reader turns pages, and checks the
# frames for a page that comes back after it was left.
#
# The window is a bare WebKitGTK 6 view, set up as the numen window is, drawn by GTK's own renderer
# in a private headless sway, so nothing appears on the desktop of the person running it. It
# loads a debug server of the e2e run:
#
#   NUMEN_TEST_PORT=3456 node scripts/start-server.mjs        (in one terminal, with apps/desktop/numen-debug built)
#   tools/epub-frames/record.sh                               (in another)
#
# Environment: NUMEN_TEST_PORT (3456), BOOK (assets/ledger.epub, a path in the vault of the run),
# OUTPUT_SCALE (1.5), RUN_NAME (turns), OUT_DIR (a new folder under /tmp), BACKEND_DELAY_MS (the delay of
# the answer that carries the markup of a chapter, none by default). Needs nix and a GPU render node.
# Exit code 1 when a page came back.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
port=${NUMEN_TEST_PORT:-3456}
book=${BOOK:-assets/ledger.epub}
scale=${OUTPUT_SCALE:-1.5}
name=${RUN_NAME:-turns}
out=${OUT_DIR:-$(mktemp -d /tmp/epub-frames.XXXXXX)}
mkdir -p "$out"

nixpkg() { nix build --no-link --print-out-paths "nixpkgs#$1" 2>/dev/null | head -1; }
sway=$(nixpkg sway)/bin/sway
swaymsg=$(nixpkg sway)/bin/swaymsg
recorder=$(nixpkg wf-recorder)/bin/wf-recorder
ffmpeg_bin=$(nixpkg ffmpeg-headless^bin)/bin
python=$(nix build --impure --no-link --print-out-paths --expr \
  'let p = (builtins.getFlake "nixpkgs").legacyPackages.x86_64-linux; in p.python3.withPackages (ps: [ ps.pygobject3 ])')/bin/python3
typelibs=""
for pkg in webkitgtk_6_0 gtk4 harfbuzz graphene libsoup_3 pango gdk-pixbuf glib cairo gobject-introspection; do
  case $pkg in pango|gdk-pixbuf|glib|cairo|gobject-introspection) selector="$pkg^*" ;; *) selector=$pkg ;; esac
  for dir in $(nix build --no-link --print-out-paths "nixpkgs#$selector" 2>/dev/null); do
    [ -d "$dir/lib/girepository-1.0" ] && typelibs="$typelibs:$dir/lib/girepository-1.0"
  done
done
networking=$(nixpkg glib-networking)/lib/gio/modules
runtime=${XDG_RUNTIME_DIR:-/run/user/$(id -u)}

cleanup() {
  kill "${browser_pid:-}" "${recorder_pid:-}" "${sway_pid:-}" 2>/dev/null || true
}
trap cleanup EXIT

before=$(ls "$runtime" | grep -E "^wayland-[0-9]+\$" || true)
printf 'output HEADLESS-1 resolution 1600x1000 position 0 0\ndefault_border none\n' > "$out/sway.conf"
env -i PATH="$PATH" HOME="$out/home" XDG_RUNTIME_DIR="$runtime" WLR_BACKENDS=headless \
  WLR_LIBINPUT_NO_DEVICES=1 "$sway" -c "$out/sway.conf" > "$out/sway.log" 2>&1 &
sway_pid=$!
until ls "$runtime"/sway-ipc.*."$sway_pid".sock > /dev/null 2>&1; do sleep 0.2; done
export SWAYSOCK=$(ls "$runtime"/sway-ipc.*."$sway_pid".sock)
for _ in $(seq 1 50); do
  display=$(comm -13 <(echo "$before") <(ls "$runtime" | grep -E "^wayland-[0-9]+\$" || true) | head -1)
  [ -n "$display" ] && break
  sleep 0.2
done
"$swaymsg" output HEADLESS-1 mode 2400x1500 scale "$scale" > /dev/null

fifo="$out/page.fifo"
rm -f "$fifo"
mkfifo "$fifo"
{ printf 'window.BOOK_PATH = "%s";\n' "$book"; cat "$here/open.mjs"; } > "$out/open.js"
env -i PATH="$PATH" HOME="$out/home" XDG_RUNTIME_DIR="$runtime" WAYLAND_DISPLAY="$display" GDK_BACKEND=wayland \
  GI_TYPELIB_PATH="${typelibs#:}" GIO_EXTRA_MODULES="$networking" MINI_FIFO="$fifo" \
  "$python" "$here/browser.py" "https://127.0.0.1:$port/" "$out/open.js" > "$out/browser.log" 2>&1 &
browser_pid=$!

ready=0
for _ in $(seq 1 300); do
  if grep -aqE 'PAGE (READY|ERR)' "$out/browser.log"; then ready=1; break; fi
  sleep 0.2
done
[ "$ready" = 1 ] && grep -aq READY "$out/browser.log" || { echo "the reader did not open"; tail -5 "$out/browser.log"; exit 2; }
grep -a AREA "$out/browser.log"
area=$(grep -a 'PAGE AREA' "$out/browser.log" | sed 's/.*AREA //')
IFS=, read -r ax ay aw ah <<< "$area"

echo "@$here/watch.mjs" > "$fifo"
if [ -n "${BACKEND_DELAY_MS:-}" ]; then
  { printf "window.BACKEND_DELAY_MS = %s;\n" "$BACKEND_DELAY_MS"; cat "$here/delay.mjs"; } > "$out/delay.js"
  echo "@$out/delay.js" > "$fifo"
fi
echo "@$here/reset.mjs" > "$fifo"
for _ in $(seq 1 300); do
  grep -aq RESET-DONE "$out/browser.log" && break
  sleep 0.2
done
sleep 0.5
start=$(date +%s%3N)
echo "$start" > "$out/$name.start"
# A strip along the left column of the reading area: the digits and the words of every page differ there.
env WAYLAND_DISPLAY="$display" "$recorder" -D -r 60 -g "$((ax + 30)),$((ay + 3)) 300x$((ah - 6))" \
  -c ffv1 -p slices=4 -p coder=0 -f "$out/$name.mkv" > "$out/recorder.log" 2>&1 &
recorder_pid=$!
sleep 1.5
echo "@$here/turns.mjs" > "$fifo"
for _ in $(seq 1 300); do
  grep -aq 'TURNS-DONE' "$out/browser.log" && break
  sleep 0.2
done
sleep 0.5
kill -INT "$recorder_pid"
wait "$recorder_pid" 2> /dev/null || true

grep -a 'PRESS' "$out/browser.log" | awk '{print $3, $4}' > "$out/$name.presses"
grep -a "PAGE OBS" "$out/browser.log" | awk '{print $3, $4}' > "$out/$name.counts"
echo "page counter after each turn:"
grep -a 'PAGE OBS' "$out/browser.log" | awk '{print $4, $5, $6}' | tr '\n' ';'
echo
PATH="$ffmpeg_bin:$PATH" node "$here/analyze.mjs" "$out/$name.mkv" "$out/$name.presses" "$start" "$out/$name.counts"
