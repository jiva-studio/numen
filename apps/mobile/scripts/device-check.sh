#!/usr/bin/env bash
# Checks the phone against the arm64 emulator by hand: whole tree, a file on tap, leaving and returning.
# Run on mridanga with the toolchain of the bbt environment. It is not run by CI.
#
# The steps are the scenarios of features/mobile/sync. Each step is a function; a step that is not written yet
# fails the run with its name.
#
# Needs: ANDROID_SDK_ROOT, JAVA_HOME, adb and emulator on PATH, the AVD numen-arm64, node with playwright,
# and a fakesync binary (tools/syncserver/cmd/fakesync) for the sync server.
set -euo pipefail

AVD="${AVD:-numen-arm64}"
SERVER_PORT="${SERVER_PORT:-18700}"
# The emulator reaches the host at 10.0.2.2. Cleartext is allowed to that address by the debug network config only.
SERVER_URL="http://10.0.2.2:${SERVER_PORT}"
PACKAGE="md.numen.spike"
WORK="$(mktemp -d)"
trap 'cleanup' EXIT

cleanup() {
  adb emu kill >/dev/null 2>&1 || true
  [[ -n "${SERVER_PID:-}" ]] && kill "${SERVER_PID}" >/dev/null 2>&1 || true
  rm -rf "${WORK}"
}

todo() {
  echo "device-check: step '$1' is not written yet" >&2
  exit 2
}

start_emulator() { todo start_emulator; }          # emulator -avd "$AVD" -no-window; adb wait-for-device
start_server() { todo start_server; }              # fakesync on SERVER_PORT with a vault "Test" holding a note and Books/Guide.pdf of 5 MB
install_debug_apk() { todo install_debug_apk; }    # make -C apps/mobile debug; adb install -r
restore_identity() { todo restore_identity; }      # debug build only: recovery words by an intent extra
connect_webview() { todo connect_webview; }        # adb forward of the webview devtools socket; playwright connectOverCDP
check_whole_tree() { todo check_whole_tree; }      # the tree lists the note and "Guide.pdf" marked "Not downloaded"
check_open_on_tap() { todo check_open_on_tap; }    # tap Guide.pdf; the marker goes away; the file shows
check_leave_stops_stream() { todo check_leave_stops_stream; } # adb shell input keyevent KEYCODE_HOME; the server counts 0 open streams
check_return_restarts_stream() { todo check_return_restarts_stream; } # adb shell monkey -p "$PACKAGE" 1; the server counts 1 open stream

main() {
  start_emulator
  start_server
  install_debug_apk
  restore_identity
  connect_webview
  check_whole_tree
  check_open_on_tap
  check_leave_stops_stream
  check_return_restarts_stream
  echo "device-check: every step passed"
}

main "$@"
