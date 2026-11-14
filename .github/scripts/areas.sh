#!/usr/bin/env bash
# Prints the areas a change touches as name=value lines for $GITHUB_OUTPUT:
#
#   areas=<json>        every filter of .github/filters.yml as true or false
#   plugin_dirs=<dirs>  the plugin folders to lint, build and test, separated by spaces
#   e2e=<scope>         the scope tests/e2e/desktop/scripts/select-tests.mjs chooses
#
# EVENT is the event name and FILTERS the outputs of dorny/paths-filter as JSON, `{}` when it did
# not run. A pull request takes the filters that matched and the changed files of the API (PULL,
# GH_TOKEN), a push the filters and the files git changed between BEFORE and HEAD. Any other event
# takes every area, every plugin and every end-to-end test.
set -euo pipefail

everything=true
if [[ $EVENT == pull_request ]]; then everything=false; fi
# BEFORE is all zeros on the first push of a branch.
if [[ $EVENT == push && ! $BEFORE =~ ^0+$ ]]; then everything=false; fi

grep -oE '^[a-z_]+:' .github/filters.yml | tr -d : | jq -R . | jq -cs \
  --argjson filters "$FILTERS" --argjson everything "$everything" '
    map({ key: ., value: ($everything or $filters[.] == "true") }) | from_entries' \
  | sed 's/^/areas=/'

list_files() {
  if [[ $EVENT == push ]]; then
    git fetch --quiet --no-tags --depth=1 origin "$BEFORE"
    git diff --name-only --no-renames "$BEFORE" HEAD
  else
    gh api --paginate "repos/$GITHUB_REPOSITORY/pulls/$PULL/files" --jq '.[].filename'
  fi
}

list_plugin_files() {
  list_files | grep -E '^plugins/[^/]+/' | grep -vE '^plugins/\.example/' || true
}

if [[ $everything == true || $(jq -r '.plugins_all' <<< "$FILTERS") == true ]]; then
  dirs=$(for manifest in plugins/*/manifest.json; do dirname "$manifest"; done)
else
  dirs=$(list_plugin_files | cut -d/ -f1-2 | sort -u | while read -r dir; do
    if [[ -f $dir/manifest.json ]]; then echo "$dir"; fi
  done)
fi
echo "plugin_dirs=$(echo "$dirs" | xargs)"

if [[ $everything == true ]]; then
  echo "e2e=all"
else
  list_files | node tests/e2e/desktop/scripts/select-tests.mjs scope | sed 's/^scope=/e2e=/'
fi
