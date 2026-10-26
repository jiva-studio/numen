#!/bin/sh
# Runs the commands the plugins declare through numen-cli over a temporary vault, with mock models
# and no network: transcribe, recognise, import and proofread, then scan, and searches for the text
# each one made.
#
#   unshare -rn sh -c 'ip link set lo up && tests/e2e/cli/plugin-commands.sh'
set -eu

repo=$(cd "$(dirname "$0")/../../.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/numen-cli-e2e.XXXXXX")
pages_port=${NUMEN_TEST_PORT:-3467}

mkdir -p "$work/bin" "$work/vault/talks" "$work/vault/books" "$work/config" "$work/cache"
(cd "$repo/apps/desktop" && go build -o "$work/bin/numen-cli" ./cmd/numen-cli)
for plugin in engines-transcription engines-ocr tools-media-importer tools-proofreading; do
	(cd "$repo/plugins/$plugin" && mkdir -p bin && go build -o "bin/$plugin" ./backend)
done

printf '# Lectures\n\nA note about the talks.\n' >"$work/vault/lectures.md"
cp "$repo/tests/e2e/desktop/fixtures/lecture.wav" "$work/vault/talks/lecture.wav"
cp "$repo/tests/e2e/desktop/fixtures/flywheel.pdf" "$work/vault/books/flywheel.pdf"

export HOME="$work" XDG_CONFIG_HOME="$work/config" XDG_CACHE_HOME="$work/cache"
export NUMEN_MOCK_ML=true NUMEN_PLUGINS_PATH="$repo/plugins"
export HTTP_PROXY="http://127.0.0.1:$pages_port" NO_PROXY=127.0.0.1

# The address the import fetches, served by a local stand-in through the proxy.
node -e '
const http = require("node:http"), fs = require("node:fs")
http.createServer((req, res) => {
  if (new URL(req.url, "http://" + req.headers.host).hostname !== "tidal-mills.numen.test") return res.writeHead(404).end()
  res.writeHead(200, { "content-type": "text/html; charset=utf-8" }).end(fs.readFileSync(process.argv[1]))
}).listen(Number(process.argv[2]), "127.0.0.1")
' "$repo/tests/e2e/desktop/fixtures/article.html" "$pages_port" &
pages=$!
trap 'kill $pages; rm -rf "$work"' EXIT

# The mock models open no model file, so each model a plugin declares is marked as already fetched.
arch=$(go env GOOS)-$(go env GOARCH)
cat "$repo"/plugins/*/manifest.json |
	jq -r '(.dependencies // [])[] | select(.kind == "model") | [.id, .metadata.fileName, .checksumSha256] | @tsv' |
	while IFS="$(printf '\t')" read -r id name sum; do
		dir="$HOME/.numen/dependencies/model/$id/latest-$arch"
		mkdir -p "$dir" && : >"$dir/$name" && printf '%s\n%s\n' "$name" "$sum" >"$dir/.ready"
	done

cli() { "$work/bin/numen-cli" --registry "$work/vaults.json" --index "$work/index.db" "$@"; }

# expect runs a command and fails unless its output holds the text given.
expect() {
	want=$1
	shift
	echo "\$ numen-cli $*"
	out=$(cli "$@" 2>&1) || { echo "$out"; exit 1; }
	echo "$out" | grep -v engines-embeddings
	case $out in
	*"$want"*) ;;
	*) echo "FAIL: numen-cli $* did not say: $want" && exit 1 ;;
	esac
}

cli vault add "$work/vault" --name talks >/dev/null
expect "numen-cli transcribe <vault> <file>" help
expect "transcribed talks/lecture.wav" transcribe talks talks/lecture.wav
expect "recognised 1 pages of books/flywheel.pdf" recognise talks books/flywheel.pdf
expect "its text fetched" import talks --address http://tidal-mills.numen.test/
expect "proofread 1 lines of talks/lecture.wav" proofread talks talks/lecture.wav
expect "3 sources: 3 read" scan talks
expect "Mock transcribed text for automated test." search talks "transcribed text"
expect "a flywheel keeps it in a rotor" search talks flywheel
expect "A tidal mill stored seawater" search talks tidal
echo "ok"
