#!/usr/bin/env bash
# Takes the areas of areas.sh (AREAS, PLUGIN_DIRS, E2E) and drops each one whose inputs already
# passed, then prints name=value lines for $GITHUB_OUTPUT:
#
#   areas=<json>, plugin_dirs=<dirs>, e2e=<scope>  as areas.sh prints them, less the areas that passed
#   marks=<json>  [{"job", "key"}]: the marker of each area that runs, and the ci.yml job running it
#
# A marker is a cache entry named passed-<area>-<hash>. The hash covers the blobs of the area's
# filter in .github/filters.yml (for e2e, of the files `select-tests.mjs inputs` prints), of ci.yml,
# the called workflow, .github/actions/ and .github/scripts/, and the values the job takes beside
# the area: PLATFORMS for go, the folders for plugin_dirs. An e2e marker also names ENGINES and the
# scope; a marker of every test, or of both engines, stands for fewer.
#
# A pull request reads the markers of its merge ref and of main, which are the caches it can
# restore; a push to main reads main's; the nightly and a manual run read none.
set -euo pipefail

filters=$(yq -o=json 'explode(.)' .github/filters.yml)
wiring=$(yq -o=json '.jobs' .github/workflows/ci.yml | jq -r '
  to_entries[] | select(.value.uses // "" | startswith("./")) | .key as $job | (.value.uses | ltrimstr("./")) as $workflow
  | .value.with // {} | to_entries[] | .value | tostring
  | (capture("outputs\\.areas\\)\\.(?<area>[a-z_]+)") // capture("outputs\\.(?<area>plugin_dirs|e2e)\\b"))
  | [$job, $workflow, .area] | @tsv')

case $EVENT in
  pull_request) refs=("refs/pull/$PULL/merge" refs/heads/main) ;;
  push) refs=("$GITHUB_REF") ;;
  *) refs=() ;;
esac
passed=$(for ref in "${refs[@]}"; do
  gh api --paginate "repos/$GITHUB_REPOSITORY/actions/caches?ref=$ref&key=passed-&per_page=100" \
    --jq '.actions_caches[].key'
done)

blobs=$RUNNER_TEMP/blobs
git ls-files -s > "$blobs"

list_pathspecs() {
  jq -r --arg area "$1" '.[$area] | flatten[]
    | if test("[{}]") then error("git pathspecs have no braces: \(.)") else . end
    | if startswith("!") then ":(glob,exclude)" + .[1:] else ":(glob)" + . end' <<< "$filters"
}

list_area_blobs() {
  local specs
  case $1 in
    e2e)
      cut -f2 "$blobs" | node tests/e2e/desktop/scripts/select-tests.mjs inputs \
        | awk -F'\t' 'NR == FNR { keep[$0]; next } $2 in keep' - "$blobs"
      ;;
    plugin_dirs)
      mapfile -t specs < <(list_pathspecs plugins_all)
      git ls-files -s -- "${specs[@]}" ':(glob)plugins/**'
      ;;
    *)
      mapfile -t specs < <(list_pathspecs "$1")
      git ls-files -s -- "${specs[@]}"
      ;;
  esac
}

get_hash() {
  local area=$1 job=$2 workflow=$3
  {
    list_area_blobs "$area"
    git ls-files -s -- .github/filters.yml .github/workflows/ci.yml "$workflow" \
      ':(glob).github/actions/**' ':(glob).github/scripts/**'
    case $job in go) echo "platforms=$PLATFORMS" ;; esac
    case $area in plugin_dirs) echo "plugin_dirs=$PLUGIN_DIRS" ;; esac
  } | sort -u | sha256sum | cut -c1-40
}

is_running() {
  case $1 in
    plugin_dirs) [[ -n $PLUGIN_DIRS ]] ;;
    e2e) [[ $E2E != none ]] ;;
    *) [[ $(jq -r --arg area "$1" '.[$area]' <<< "$AREAS") == true ]] ;;
  esac
}

engines=${ENGINES// /-}
scope=$(printf '%s' "$E2E" | sha256sum | cut -c1-16)
areas=$AREAS
plugin_dirs=$PLUGIN_DIRS
e2e=$E2E
marks='[]'
while IFS=$'\t' read -r job workflow area; do
  if ! is_running "$area"; then continue; fi
  hash=$(get_hash "$area" "$job" "$workflow")
  key=passed-$area-$hash
  accepted=("$key")
  if [[ $area == e2e ]]; then
    key=$key-$engines-$scope
    accepted=("$key" "passed-e2e-$hash-$engines-all" "passed-e2e-$hash-chromium-webkit-$scope" "passed-e2e-$hash-chromium-webkit-all")
  fi
  marker=$(grep -Fxm1 -f <(printf '%s\n' "${accepted[@]}") <<< "$passed" || true)
  if [[ -z $marker ]]; then
    echo "runs $area: no $key" >&2
    marks=$(jq -c --arg job "$job" --arg key "$key" '. + [{job: $job, key: $key}]' <<< "$marks")
    continue
  fi
  echo "skips $area: $marker" >&2
  case $area in
    plugin_dirs) plugin_dirs= ;;
    e2e) e2e=none ;;
    *) areas=$(jq -c --arg area "$area" '.[$area] = false' <<< "$areas") ;;
  esac
done <<< "$wiring"

echo "areas=$areas"
echo "plugin_dirs=$plugin_dirs"
echo "e2e=$e2e"
echo "marks=$marks"
