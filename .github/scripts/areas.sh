#!/usr/bin/env bash
# Prints the areas a change touches as name=value lines for $GITHUB_OUTPUT:
#
#   areas=<json>        every filter of .github/filters.yml as true or false, and docs_deploy
#   plugin_dirs=<dirs>  the plugin folders to lint, build and test, separated by spaces
#   e2e=<scope>         the scope tests/e2e/desktop/scripts/select-tests.mjs chooses
#
# EVENT is the event name and FILTERS the outputs of dorny/paths-filter as JSON, `{}` when it did
# not run. A pull request takes the filters that matched; any other event takes every area, every
# plugin and every end-to-end test. docs_deploy is true on a push that changed the manual.
set -euo pipefail

everything=true
if [ "$EVENT" = pull_request ]; then everything=false; fi

grep -oE '^[a-z_]+:' .github/filters.yml | tr -d : | jq -R . | jq -cs \
  --argjson filters "$FILTERS" --argjson everything "$everything" --arg event "$EVENT" '
    map({ key: ., value: ($everything or $filters[.] == "true") }) | from_entries
    | .docs_deploy = ($event == "push" and $filters.docs == "true")' \
  | sed 's/^/areas=/'

list_files() {
  jq -r --arg name "$1" '.[$name + "_files"] // "[]" | fromjson | .[]' <<< "$FILTERS"
}

if [ "$everything" = true ] || [ "$(jq -r '.plugins_all' <<< "$FILTERS")" = true ]; then
  dirs=$(for manifest in plugins/*/manifest.json; do dirname "$manifest"; done)
else
  dirs=$(list_files plugin_files | cut -d/ -f1-2 | sort -u | while read -r dir; do
    if [ -f "$dir/manifest.json" ]; then echo "$dir"; fi
  done)
fi
echo "plugin_dirs=$(echo "$dirs" | xargs)"

if [ "$everything" = true ]; then
  echo "e2e=all"
else
  list_files files | node tests/e2e/desktop/scripts/select-tests.mjs scope | sed 's/^scope=/e2e=/'
fi
