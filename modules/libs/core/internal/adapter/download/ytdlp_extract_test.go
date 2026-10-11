package download

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/operation"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

const anExtractedVideo = `{"title":"Entropy explained","language":"en","duration":83.5,` +
	`"subtitles":{"en":[{}]}}`

const anAddress = domain.URL("https://youtu.be/dQw4w9WgXcQ")

// recordingTool is a yt-dlp that appends each invocation's arguments to a log,
// answers the version, the info dump, subtitles and a copy, and refuses a
// --load-info-json file that is not there. versionBody is the shell run for
// --version.
func recordingTool(t *testing.T, versionBody string) (command []string, invocations func() []string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the tool is a shell script")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	script := `#!/bin/sh
echo "$*" >> '` + log + `'
out=""; prev=""
for a in "$@"; do
  if [ "$prev" = "-o" ]; then out="$a"; fi
  if [ "$prev" = "--load-info-json" ] && [ ! -f "$a" ]; then echo "no such info" >&2; exit 2; fi
  prev="$a"
done
case "$*" in
  *--version*) ` + versionBody + ` ;;
  *--dump-single-json*) echo '` + anExtractedVideo + `' ;;
  *--write-subs*) printf '{"events":[{"tStartMs":0,"dDurationMs":2000,"segs":[{"utf8":"The first line."}]}]}' > "$out.en.json3" ;;
  *) printf x > "$(echo "$out" | sed 's/%(ext)s/mp4/')" ;;
esac
`
	at := filepath.Join(dir, "yt-dlp")
	if err := os.WriteFile(at, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return []string{at}, func() []string {
		raw, err := os.ReadFile(log)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			t.Fatal(err)
		}
		return strings.Split(strings.TrimSpace(string(raw)), "\n")
	}
}

func countInvocations(all []string, flag string) int {
	n := 0
	for _, one := range all {
		if strings.Contains(one, flag) {
			n++
		}
	}
	return n
}

// useEmptyTempDir points the temporary folder at an empty one and answers a
// function listing what is left in it.
func useEmptyTempDir(t *testing.T) func() []string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	return func() []string {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, one := range entries {
			names = append(names, one.Name())
		}
		return names
	}
}

func newTestYtDLP(command []string) *ytDLP {
	return newYtDLP(Config{Video: Tool{Command: command}})
}

func TestTextExtractsAnAddressOnce(t *testing.T) {
	left := useEmptyTempDir(t)
	command, invocations := recordingTool(t, "echo 2099.01.01")
	v := newTestYtDLP(command)

	got, err := v.Text(t.Context(), anAddress, port.PreferredCaptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Entropy explained" || len(got.Cues) != 1 {
		t.Errorf("text reads %+v", got)
	}
	all := invocations()
	if n := countInvocations(all, "--dump-single-json"); n != 1 {
		t.Errorf("the address was extracted %d times: %q", n, all)
	}
	if n := countInvocations(all, "--load-info-json"); n != 1 {
		t.Errorf("the subtitles run loaded the saved info %d times: %q", n, all)
	}
	if rest := left(); len(rest) != 0 {
		t.Errorf("left behind: %v", rest)
	}
}

func TestACopyExtractsAnAddressOnce(t *testing.T) {
	left := useEmptyTempDir(t)
	command, invocations := recordingTool(t, "echo 2099.01.01")
	v := newTestYtDLP(command)

	ctx, end := operation.Begin(t.Context())
	if _, err := v.Metadata(ctx, anAddress); err != nil {
		t.Fatal(err)
	}
	var into strings.Builder
	copied, err := v.Download(ctx, anAddress, &into)
	if err != nil {
		t.Fatal(err)
	}
	if copied.Extension != ".mp4" || into.String() != "x" {
		t.Errorf("copy reads %+v with %q", copied, into.String())
	}
	if rest := left(); len(rest) != 1 {
		t.Errorf("the saved info is not held until the operation ends: %v", rest)
	}
	end()

	all := invocations()
	if n := countInvocations(all, "--dump-single-json"); n != 1 {
		t.Errorf("the address was extracted %d times: %q", n, all)
	}
	if n := countInvocations(all, "--load-info-json"); n != 1 {
		t.Errorf("the download loaded the saved info %d times: %q", n, all)
	}
	if rest := left(); len(rest) != 0 {
		t.Errorf("left behind: %v", rest)
	}
}

// Outside an operation nothing is kept between calls.
func TestMetadataOutsideAnOperationExtractsEachTime(t *testing.T) {
	left := useEmptyTempDir(t)
	command, invocations := recordingTool(t, "echo 2099.01.01")
	v := newTestYtDLP(command)

	for range 2 {
		if _, err := v.Metadata(t.Context(), anAddress); err != nil {
			t.Fatal(err)
		}
	}
	if n := countInvocations(invocations(), "--dump-single-json"); n != 2 {
		t.Errorf("the address was extracted %d times, want 2", n)
	}
	if rest := left(); len(rest) != 0 {
		t.Errorf("left behind: %v", rest)
	}
}

func TestAFailedExtractionIsReportedAndLeavesNothing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the tool is a shell script")
	}
	left := useEmptyTempDir(t)
	dir := t.TempDir()
	at := filepath.Join(dir, "yt-dlp")
	if err := os.WriteFile(at, []byte("#!/bin/sh\necho 'ERROR: Sign in to confirm you are not a bot' >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	v := newTestYtDLP([]string{at})

	_, err := v.Text(t.Context(), anAddress, port.PreferredCaptions{})
	if err == nil || !strings.Contains(err.Error(), "not a bot") {
		t.Errorf("the refusal reads %v", err)
	}
	if rest := left(); len(rest) != 0 {
		t.Errorf("left behind: %v", rest)
	}
}

func TestTheVersionIsNotReadAtConstructionAndIsReadOnce(t *testing.T) {
	command, invocations := recordingTool(t, "echo 2099.01.01")
	v := newTestYtDLP(command)

	if all := invocations(); len(all) != 0 {
		t.Fatalf("constructing ran the tool: %q", all)
	}
	for range 3 {
		if got := v.GetDownloadModel(anAddress).Version; got != "2099.01.01" {
			t.Errorf("version reads %q", got)
		}
	}
	if n := countInvocations(invocations(), "--version"); n != 1 {
		t.Errorf("the version was read %d times, want once", n)
	}
}

func TestAHangingVersionProbeTimesOut(t *testing.T) {
	command, _ := recordingTool(t, "sleep 600")
	v := newTestYtDLP(command)
	v.probeTimeout = 100 * time.Millisecond

	began := time.Now()
	got := v.GetDownloadModel(anAddress)
	if got.Version != "" {
		t.Errorf("version reads %q, want none", got.Version)
	}
	if got.Tool != "yt-dlp" || got.Producer == "" {
		t.Errorf("model reads %+v", got)
	}
	if took := time.Since(began); took > time.Minute {
		t.Errorf("the probe took %v", took)
	}
}
