#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../.."
temp=$(mktemp -d)
trap 'rm -r "$temp"' EXIT

cat > "$temp/docker" <<'DOCKER'
#!/usr/bin/env bash
set -euo pipefail
case "$1" in
  compose) printf '%s %s\n' "$6" "$3" >> "$STACK_TEST_CALLS" ;;
  run) printf '%s\n' "${4%%:*}" ;;
  *) exit 2 ;;
esac
DOCKER
chmod +x "$temp/docker"
export PATH="$temp:$PATH" STACK_TEST_CALLS="$temp/calls"
export NUMEN_SYNC_IMAGE=test NUMEN_SYNC_DEVTOOLS_IMAGE=test NUMEN_SYNC_PORT_BASE=18787

check_project() {
  local requested=$1 project=$2 settings
  : > "$STACK_TEST_CALLS"
  make -s sync_stack_up PROJECT="$requested"
  make -s sync_stack_down PROJECT="$requested"
  diff -u <(printf 'up %s\ndown %s\n' "$project" "$project") "$STACK_TEST_CALLS"
  settings=$(make -s sync_stack_env PROJECT="$requested")
  diff -u <(bash tests/server/stack.sh env "$requested") <(printf '%s\n' "$settings")
  eval "$settings"
  [[ $NUMEN_SYNC_TEST_ROOT_KEY == "${project}_keys" ]]
  [[ $NUMEN_SYNC_TEST_URL == http://127.0.0.1:18787 ]]
  [[ $NUMEN_SYNC_TEST_PAYMENT_URL == http://127.0.0.1:18789 ]]
  [[ $NUMEN_SYNC_TEST_BUCKET_URL == http://127.0.0.1:18790 ]]
}

check_project numen-sync-other numen-sync-other
check_project "" numen-sync-test
