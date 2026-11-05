#!/usr/bin/env bash
# Prints `matrix=<json>`, the machines the core tests run on, for $GITHUB_OUTPUT. Linux always;
# Windows and macOS when a commit of the change carries a `Run-On:` trailer naming them (or `all`),
# or when a manual run asks for them (ASKED_WINDOWS, ASKED_MACOS). The commits of a pull request
# are read from the API (PULL, GH_TOKEN), those of a push from the event.
set -euo pipefail

windows=false
macos=false
case "$EVENT" in
  workflow_dispatch) windows=$ASKED_WINDOWS; macos=$ASKED_MACOS; messages= ;;
  pull_request) messages=$(gh api --paginate "repos/$GITHUB_REPOSITORY/pulls/$PULL/commits" --jq '.[].commit.message') ;;
  *) messages=$(jq -r '.commits[]?.message' "$GITHUB_EVENT_PATH") ;;
esac
asked=$(printf '%s\n' "$messages" | grep -i '^Run-On:' | cut -d: -f2- \
  | tr '[:upper:]' '[:lower:]' | tr -d ' \t\r' | tr '\n' ',' || true)
echo "asked: ${asked:-nothing}" >&2
case ",$asked," in *,all,*) windows=true; macos=true ;; esac
case ",$asked," in *,windows,*) windows=true ;; esac
case ",$asked," in *,macos,*) macos=true ;; esac

# Windows and macOS are reported and do not hold the change. What stands open on macOS:
# adapter/cli and adapter/mcp compare a vault's folder against the path a test made it at, and a
# mac resolves /tmp to /private/tmp; flashcards parts from the library on a card face a day ahead
# of its own last answer.
#
# race is the detector: Linux runs it, and Windows runs the same tests without it on the slowest
# runner. minutes is what the suite is given.
entries='{"name":"Linux","runner":"ubuntu-latest","optional":false,"race":true,"minutes":45}'
if [[ $windows == true ]]; then
  entries="$entries,"'{"name":"Windows","runner":"windows-latest","optional":true,"race":false,"minutes":15}'
fi
if [[ $macos == true ]]; then
  entries="$entries,"'{"name":"macOS","runner":"macos-latest","optional":true,"race":true,"minutes":45}'
fi
echo "matrix=[$entries]"
