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
	if env := getEnvironment(fetch.Tool{}, []string{"BASE=1"}); env != nil {
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

// A failed run wraps the process exit error, or returns the context error when the context is canceled.
func TestRunWrapsExecutionError(t *testing.T) {
	_, err := run(t.Context(), process.Runner{}, program{command: []string{"sh", "-c", `echo failed >&2; exit 2`}}, nil)
	if err == nil {
		t.Fatal("run did not fail")
	}
	if !errors.As(err, new(*exec.ExitError)) {
		t.Errorf("error %v does not wrap *exec.ExitError", err)
	}
	canceledCtx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := run(canceledCtx, &stubRunner{runErr: errors.New("killed")}, program{command: []string{"sh"}}, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("run with canceled context = %v, want context.Canceled", err)
	}
}

type stubRunner struct {
	environCalls int
	runErr       error
}

func (r *stubRunner) LookPath(file string) (string, error) { return "/usr/bin/" + file, nil }

func (r *stubRunner) Environ() []string {
	r.environCalls++
	return []string{"BASE=x"}
}

func (r *stubRunner) Run(context.Context, []string, []string, io.Writer, ...string) ([]byte, string, error) {
	return nil, "failed", r.runErr
}

func (r *stubRunner) ReportError(string, error) {}

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
	if got := resolveProgram(fetch.Tool{}, "sh", nil); got.isPresent() {
		t.Errorf("nil runner resolved to %v", got.command)
	}
	if got := resolveProgram(fetch.Tool{Command: []string{"/bin/sh"}, Environment: map[string]string{"A": "1"}}, "sh", nil); !slices.Equal(got.env, []string{"A=1"}) {
		t.Errorf("nil runner with env = %v, want [A=1]", got.env)
	}
	sr := &stubRunner{}
	if got := resolveProgram(fetch.Tool{Command: []string{"/bin/sh"}}, "sh", sr); got.env != nil || sr.environCalls != 0 {
		t.Errorf("empty env called Environ %d times, env=%v", sr.environCalls, got.env)
	}
	if got := resolveProgram(fetch.Tool{Command: []string{"/bin/sh"}, Environment: map[string]string{"A": "1"}}, "sh", sr); !slices.Equal(got.env, []string{"BASE=x", "A=1"}) || sr.environCalls != 1 {
		t.Errorf("non-empty env = %v, environCalls=%d", got.env, sr.environCalls)
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
	if _, err := empty.findPlayer("https://example.com"); !errors.Is(err, ErrNoTool) {
		t.Errorf("playerFor error = %v, want ErrNoTool", err)
	}
}

type fakeWeb struct {
	page     string
	err      error
	readErr  error
	closeErr error
}

type fakePageReader struct {
	io.Reader
	readErr  error
	closeErr error
}

func (r fakePageReader) Read(p []byte) (int, error) {
	if r.readErr != nil {
		return 0, r.readErr
	}
	return r.Reader.Read(p)
}

func (r fakePageReader) Close() error { return r.closeErr }

func (w fakeWeb) Fetch(context.Context, string) (io.ReadCloser, error) {
	if w.err != nil {
		return nil, w.err
	}
	return fakePageReader{Reader: strings.NewReader(w.page), readErr: w.readErr, closeErr: w.closeErr}, nil
}

func TestPagesProviderGetsTheArticleText(t *testing.T) {
	validHTML := "<html><head><title>T</title></head><body><article><p>Readability body text here.</p></article></body></html>"
	pages := NewPages(fakeWeb{page: validHTML})
	if !pages.CanHandle("https://example.com/article") || pages.CanHandle("https://www.youtube.com/watch?v=tidalmill01") {
		t.Fatal("PageProvider.CanHandle misclassified site")
	}
	if isVideoSite("http://\x00") {
		t.Fatal("invalid URL treated as video site")
	}
	y := newYtDLP(fetch.Config{Video: fetch.Tool{Command: []string{"sh", "-c", "echo 2025.01.01"}}}, process.Runner{}, nil, nil)
	if y.probeTimeout != 2*time.Minute {
		t.Errorf("probeTimeout = %v, want 2m", y.probeTimeout)
	}
	if got := y.GetDownloadModel("https://youtu.be/abc").Version; got != "2025.01.01" {
		t.Errorf("ytDLP version = %q, want 2025.01.01", got)
	}
	text, err := pages.GetText(t.Context(), "https://example.com/post", fetch.PreferredCaptions{})
	if err != nil {
		t.Fatal(err)
	}
	if text.Title != "T" || text.Producer != "article" || text.Length != 0 || !strings.Contains(text.Prose, "Readability body text here.") {
		t.Errorf("GetText = %+v, want title T, producer article, length 0, and the body text", text)
	}
	if _, err := pages.GetText(t.Context(), "://bad-url", fetch.PreferredCaptions{}); err == nil || errors.Is(err, fetch.ErrNothingDownloaded) {
		t.Errorf("GetText(bad url) = %v, want URL parse error", err)
	}
	fetchErr := errors.New("fetch failed")
	if _, err := NewPages(fakeWeb{err: fetchErr}).GetText(t.Context(), "https://example.com/post", fetch.PreferredCaptions{}); !errors.Is(err, fetchErr) {
		t.Errorf("GetText(fetchErr) = %v, want %v", err, fetchErr)
	}
	closeErr := errors.New("close failed")
	if _, err := NewPages(fakeWeb{page: validHTML, closeErr: closeErr}).GetText(t.Context(), "https://example.com/post", fetch.PreferredCaptions{}); !errors.Is(err, closeErr) {
		t.Errorf("GetText(closeErr) = %v, want %v", err, closeErr)
	}
	if _, err := NewPages(fakeWeb{readErr: errors.New("read failed")}).GetText(t.Context(), "https://example.com/post", fetch.PreferredCaptions{}); !errors.Is(err, fetch.ErrNothingDownloaded) {
		t.Errorf("GetText(readErr) = %v, want fetch.ErrNothingDownloaded", err)
	}
	empty := fakeWeb{page: "<html><head><title>Empty</title></head><body>   </body></html>"}
	if _, err := NewPages(empty).GetText(t.Context(), "https://example.com/post", fetch.PreferredCaptions{}); !errors.Is(err, fetch.ErrNothingDownloaded) {
		t.Errorf("GetText(empty prose) = %v, want fetch.ErrNothingDownloaded", err)
	}
}

type certificateRunner struct{ stubRunner }

func (certificateRunner) Run(context.Context, []string, []string, io.Writer, ...string) ([]byte, string, error) {
	return nil, "ERROR: [youtube] x: Unable to download webpage: [SSL: CERTIFICATE_VERIFY_FAILED] certificate verify failed", errors.New("exit status 1")
}

func TestRunNamesAnUntrustedCertificate(t *testing.T) {
	_, err := run(t.Context(), &certificateRunner{}, program{command: []string{"yt-dlp"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "does not trust the certificate") || !strings.Contains(err.Error(), "CERTIFICATE_VERIFY_FAILED") {
		t.Fatalf("error = %v", err)
	}
	if _, err := run(t.Context(), &stubRunner{runErr: errors.New("exit status 1")}, program{command: []string{"yt-dlp"}}, nil); err == nil || strings.Contains(err.Error(), "certificate") {
		t.Fatalf("error = %v", err)
	}
}
