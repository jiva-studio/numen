#!/usr/bin/env bash
# Collects CPU profiles from the benchmarks of the paths the desktop spends its time in,
# merges them, and writes the result to cmd/numen/default.pgo, which `go build` picks up
# through -pgo=auto. Run it on an idle machine.
# `pgo.sh --check` fails when the committed profile names a local path.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"

# Fails when the profile names a local path of the machine that collected it.
check_profile() {
	local file="$1" text
	text="$(go tool pprof -raw "$file" 2>/dev/null)"
	if grep -qE '/home/|/Users/|(^|[^A-Za-z0-9_])[A-Za-z]:\\' <<<"$text"; then
		echo "$file contains a local path; run build/pgo.sh to normalise it" >&2
		return 1
	fi
}

if [[ "${1:-}" = "--check" ]]; then
	check_profile "$here/../cmd/numen/default.pgo"
	exit
fi

root="$(cd "$(dirname "$0")/../../../.." && pwd)"
core="$root/modules/libs/core"
out="$root/modules/apps/desktop/cmd/numen/default.pgo"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# Edges and nodes under this share of the total are dropped, which keeps the file small.
node_fraction="${PGO_NODE_FRACTION:-0.005}"

echo "load average: $(cut -d' ' -f1-3 /proc/loadavg 2>/dev/null || uptime)"

profile() {
	local name="$1" package="$2"
	shift 2
	(cd "$core" && go test "$package" -run XXX -cpuprofile "$work/$name.pprof" -o "$work/$name.test" -timeout 60m "$@" >/dev/null)
}

profile cold-scan ./usecase/vault/ -bench 'ColdScan/10000' -benchtime 3x
profile warm-scan ./usecase/vault/ -bench 'WarmScan|Incremental|Search' -benchtime 20x
profile links ./usecase/vault/ -bench 'Links|Backlinks' -benchtime 300x
profile front-door ./adapter/window/flashcards/ -bench FrontDoor -benchtime 5x
profile flashcards ./usecase/flashcards/ -bench Vault -benchtime 5x
profile search ./usecase/search/ -bench Search -benchtime 20x
profile markdown ./markdown/ -bench . -benchtime 2s
profile note ./usecase/note/ -bench . -benchtime 2s

(cd "$core" && go tool pprof -proto -nodefraction="$node_fraction" -edgefraction="$node_fraction" "$work"/*.pprof >"$work/merged.pprof" 2>/dev/null)
# File names become repository-relative, $GOROOT/src and $GOMODCACHE paths.
(cd "$here/pgonorm" && go run . "$root" "$work/merged.pprof" "$work/clean.pprof")
check_profile "$work/clean.pprof"
mv "$work/clean.pprof" "$out"
ls -l "$out"
