package download

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/process"
)

// A program run carries the environment it is given.
func TestProgramBuildCommandSetsEnvironment(t *testing.T) {
	if env := getEnvironment(fetch.Tool{}, nil); env != nil {
		t.Errorf("empty tool environment = %v, want nil", env)
	}
	configured := getEnvironment(fetch.Tool{Environment: map[string]string{"NUMEN_K": "val"}}, []string{"BASE=1"})
	if !slices.Contains(configured, "NUMEN_K=val") || !slices.Contains(configured, "BASE=1") {
		t.Errorf("configured environment missing NUMEN_K=val or BASE=1: %v", configured)
	}
	out, err := run(t.Context(), process.Runner{}, program{
		command: []string{"sh", "-c", `printf '%s' "$NUMEN_TEST_ENV"`},
		env:     []string{"NUMEN_TEST_ENV=passed_value"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "passed_value" {
		t.Errorf("run with environment returned %q, want %q", string(out), "passed_value")
	}
}

// A run directs standard output to the given writer when provided.
func TestRunRoutesOutputToCustomWriter(t *testing.T) {
	var into bytes.Buffer
	out, err := run(t.Context(), process.Runner{}, program{command: []string{"sh", "-c", `printf 'streamed'`}}, &into)
	if err != nil {
		t.Fatal(err)
	}
	if into.String() != "streamed" {
		t.Errorf("into buffer received %q, want %q", into.String(), "streamed")
	}
	if len(out) != 0 {
		t.Errorf("run output returned %q, want empty bytes when into is provided", string(out))
	}
}

// A failed run wraps the process exit error.
func TestRunWrapsExecutionError(t *testing.T) {
	_, err := run(t.Context(), process.Runner{}, program{command: []string{"sh", "-c", `echo failed >&2; exit 2`}}, nil)
	if err == nil {
		t.Fatal("run did not fail")
	}
	if !errors.As(err, new(*exec.ExitError)) {
		t.Errorf("error %v does not wrap *exec.ExitError", err)
	}
}

// A program that produces no output reports that it printed nothing.
func TestGetLastLine_ReturnsFallbackWhenEmpty(t *testing.T) {
	if got := getLastLine(""); got != "it said nothing" {
		t.Errorf("getLastLine(\"\") = %q, want %q", got, "it said nothing")
	}
	if got := getLastLine("   \n\t  \n  "); got != "it said nothing" {
		t.Errorf("getLastLine(whitespace) = %q, want %q", got, "it said nothing")
	}
	if got := getLastLine("first\nsecond\n"); got != "second" {
		t.Errorf("getLastLine(multiline) = %q, want %q", got, "second")
	}
}

func TestResolveProgramPrecedence(t *testing.T) {
	if got := resolveProgram(fetch.Tool{Command: []string{"/setting/tool"}}, "sh", process.Runner{}).command[0]; got != "/setting/tool" {
		t.Errorf("setting path = %q, want /setting/tool", got)
	}
	if got := resolveProgram(fetch.Tool{}, "sh", process.Runner{}); !got.isPresent() || got.command[0] == "sh" {
		t.Errorf("PATH lookup = %v, want the sh found on PATH", got.command)
	}
	if got := resolveProgram(fetch.Tool{}, "no-such-numen-tool", process.Runner{}); got.isPresent() {
		t.Errorf("missing tool resolved to %v", got.command)
	}
}

func TestEmptyDownloaderRejectsAllOperationsWithNoTool(t *testing.T) {
	empty := &Downloader{}
	if _, err := empty.GetMetadata(t.Context(), "https://example.com"); !errors.Is(err, ErrNoTool) {
		t.Errorf("GetMetadata error = %v, want ErrNoTool", err)
	}
	if _, err := empty.GetText(t.Context(), "https://example.com", fetch.PreferredCaptions{}); !errors.Is(err, ErrNoTool) {
		t.Errorf("GetText error = %v, want ErrNoTool", err)
	}
	if _, err := empty.Download(t.Context(), "https://example.com", fetch.MediaVideo, io.Discard); !errors.Is(err, ErrNoTool) {
		t.Errorf("Download error = %v, want ErrNoTool", err)
	}
	if _, err := empty.playerFor("https://example.com"); !errors.Is(err, ErrNoTool) {
		t.Errorf("playerFor error = %v, want ErrNoTool", err)
	}
}

type fakeWeb struct {
	page string
}

func (w fakeWeb) Fetch(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(w.page)), nil
}

func TestPagesProviderGetsTheArticleText(t *testing.T) {
	pages := NewPages(fakeWeb{page: "<html><head><title>T</title></head><body><article><p>Readability body text here.</p></article></body></html>"})
	if !pages.CanHandle("https://example.com/article") || pages.CanHandle("https://www.youtube.com/watch?v=tidalmill01") {
		t.Fatal("PageProvider.CanHandle misclassified site")
	}
	if isVideoSite("http://\x00") {
		t.Fatal("invalid URL treated as video site")
	}
	if y := newYtDLP(fetch.Config{}, process.Runner{}, nil, nil); y.probeTimeout != 2*time.Minute {
		t.Errorf("probeTimeout = %v, want 2m", y.probeTimeout)
	}
	text, err := pages.GetText(t.Context(), "https://example.com/post", fetch.PreferredCaptions{})
	if err != nil {
		t.Fatal(err)
	}
	if text.Title != "T" || !strings.Contains(text.Prose, "Readability body text here.") {
		t.Errorf("GetText = %+v, want title T and the body text", text)
	}
}
