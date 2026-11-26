#!/usr/bin/env bash
# Builds what the end-to-end tests run, in the root of a checkout whose UI libraries are built:
# the plugins, and the editor and flashcards windows with the debug binary that embeds each. The
# parts run side by side; the log of each is printed once all are done, with the seconds it took.
#
# BUILD_FLASHCARDS=true  also build the flashcards window and its binary
set -uo pipefail

start=$(date +%s)
logs=${RUNNER_TEMP:-/tmp}

timed() {
  local name=$1 began status=0
  shift
  began=$(date +%s)
  "$@" || status=$?
  echo "TIMING $name began=$((began - start)) took=$(($(date +%s) - began)) status=$status"
  return "$status"
}

run_part() {
  case $1 in
    plugins)
      timed plugins make plugins_build SKIP_UI_BUILD=1
      ;;
    editor)
      timed vite-editor npm run build --prefix apps/desktop/editor &&
        timed go-numen-debug go build -C apps/desktop -tags numen_debug -o numen-debug ./cmd/numen &&
        mkdir -p plugins/connections-llm/bin &&
        timed go-fakeopenai go build -C plugins/connections-llm -o bin/fakeopenai ./backend/cmd/fakeopenai

      ;;
    flashcards)
      timed vite-flashcards npm run build --prefix apps/desktop/flashcards &&
        timed go-flashcards-debug go build -C apps/desktop -tags numen_debug -o numen-flashcards-debug ./cmd/numen-flashcards
      ;;
  esac
}

parts=(plugins editor)
if [ "${BUILD_FLASHCARDS:-}" = true ]; then parts+=(flashcards); fi

pids=()
for part in "${parts[@]}"; do
  run_part "$part" > "$logs/build-$part.log" 2>&1 &
  pids+=($!)
done

status=0
for index in "${!parts[@]}"; do
  wait "${pids[index]}" || status=$?
  echo "::group::${parts[index]}"
  cat "$logs/build-${parts[index]}.log"
  echo "::endgroup::"
done
exit "$status"
