#!/usr/bin/env bash
# Brings the test stack of the real sync server up and down, and prints the settings the tests read.
# usage: stack.sh up | down [PROJECT] | env | fresh PROJECT [COMPOSE_FILE]
# The four published ports are NUMEN_SYNC_PORT_BASE (default 8787) and the next three: the server, the payment page, the payment control, the bucket. A stack beside another takes a different project name and base.
# fresh drops the containers and volumes of the named project, then brings it up again from COMPOSE_FILE (default: compose.yaml here) and prints its settings.
# The real server cannot delete uploaded files, so a stack that ran a full test run holds gigabytes in its bucket; fresh empties it.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
project=numen-sync-test
file=$here/compose.yaml

# The newest images of the private packages, unless the environment names others.
pull_images() {
  if [ -n "${NUMEN_SYNC_IMAGE:-}" ] || [ -n "${NUMEN_SYNC_DEVTOOLS_IMAGE:-}" ]; then
    echo "using the images named in the environment; nothing is pulled" >&2
    return
  fi
  if ! compose pull --quiet; then
    echo "::error title=No access to the server images::The images cannot be pulled. Add the repository jiva-studio/numen with Read in the package settings of both numen-sync and numen-sync-devtools (Packages, package settings, Manage Actions access), or log in to ghcr.io with a token that can read them." >&2
    exit 1
  fi
}

base=${NUMEN_SYNC_PORT_BASE:-8787}
export NUMEN_SYNC_PORT_SERVER=$base NUMEN_SYNC_PORT_PAYMENT=$((base + 1)) NUMEN_SYNC_PORT_CONTROL=$((base + 2)) NUMEN_SYNC_PORT_BUCKET=$((base + 3))

compose() { docker compose -p "$project" -f "$file" "$@"; }

print_env() {
  local root
  root=$(docker run --rm -v "${project}_keys:/keys:ro" --entrypoint cat "${NUMEN_SYNC_DEVTOOLS_IMAGE:-ghcr.io/jiva-studio/numen-sync-devtools:main}" /keys/root-public.hex)
  echo "export NUMEN_SYNC_TEST_URL=http://127.0.0.1:$NUMEN_SYNC_PORT_SERVER"
  echo "export NUMEN_SYNC_TEST_ROOT_KEY=$root"
  echo "export NUMEN_SYNC_TEST_PAYMENT_URL=http://127.0.0.1:$NUMEN_SYNC_PORT_CONTROL"
  echo "export NUMEN_SYNC_TEST_BUCKET_URL=http://127.0.0.1:$NUMEN_SYNC_PORT_BUCKET"
}

case "${1:-}" in
  up)
    pull_images
    compose up --detach --wait --quiet-pull
    ;;
  down)
    project=${2:-$project}
    compose down --volumes
    ;;
  env)
    print_env
    ;;
  fresh)
    if [ -z "${2:-}" ]; then
      echo "fresh needs the project name: stack.sh fresh PROJECT [COMPOSE_FILE]" >&2
      exit 2
    fi
    project=$2
    file=${3:-$file}
    pull_images
    echo "removing the containers and volumes of the project $project:" >&2
    compose ps --all --format '  container {{.Name}} ({{.Service}})' >&2
    docker volume ls --filter "label=com.docker.compose.project=$project" --format '  volume {{.Name}}' >&2
    compose down --volumes --remove-orphans
    compose up --detach --wait --quiet-pull
    print_env
    ;;
  *)
    echo "usage: stack.sh up | down [PROJECT] | env | fresh PROJECT [COMPOSE_FILE]" >&2
    exit 2
    ;;
esac
