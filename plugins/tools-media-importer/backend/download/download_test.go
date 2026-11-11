package download

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/process"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type paddedHTMLReader struct {
	prefix string
	pos    int
	total  int
}

func (r *paddedHTMLReader) Read(p []byte) (int, error) {
	if r.pos >= r.total {
		return 0, io.EOF
	}
	limit := min(len(p), r.total-r.pos)
	for i := range limit {
		idx := r.pos + i
		if idx < len(r.prefix) {
			p[i] = r.prefix[idx]
		} else {
			p[i] = ' '
		}
	}
	r.pos += limit
	return limit, nil
}

func (r *paddedHTMLReader) Close() error { return nil }

// A program run carries the environment it is given.
func TestProgramBuildCommandSetsEnvironment(t *testing.T) {
	if env := (Tool{}).getEnvironment(nil); env != nil {
		t.Errorf("empty tool environment = %v, want nil", env)
	}
	configured := (Tool{Environment: map[string]string{"NUMEN_K": "val"}}).getEnvironment([]string{"BASE=1"})
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
	if got := resolveProgram(Tool{Command: []string{"/setting/tool"}}, "sh", process.Runner{}).command[0]; got != "/setting/tool" {
		t.Errorf("setting path = %q, want /setting/tool", got)
	}
	if got := resolveProgram(Tool{}, "sh", process.Runner{}); !got.isPresent() || got.command[0] == "sh" {
		t.Errorf("PATH lookup = %v, want the sh found on PATH", got.command)
	}
	if got := resolveProgram(Tool{}, "no-such-numen-tool", process.Runner{}); got.isPresent() {
		t.Errorf("missing tool resolved to %v", got.command)
	}
}

func TestEmptyDownloaderRejectsAllOperationsWithNoTool(t *testing.T) {
	empty := &Downloader{}
	if _, err := empty.GetMetadata(t.Context(), "https://example.com"); !errors.Is(err, ErrNoTool) {
		t.Errorf("GetMetadata error = %v, want ErrNoTool", err)
	}
	if _, err := empty.GetText(t.Context(), "https://example.com", PreferredCaptions{}); !errors.Is(err, ErrNoTool) {
		t.Errorf("GetText error = %v, want ErrNoTool", err)
	}
	if _, err := empty.Download(t.Context(), "https://example.com", MediaVideo, io.Discard); !errors.Is(err, ErrNoTool) {
		t.Errorf("Download error = %v, want ErrNoTool", err)
	}
	if _, err := empty.playerFor("https://example.com"); !errors.Is(err, ErrNoTool) {
		t.Errorf("playerFor error = %v, want ErrNoTool", err)
	}
}

func TestPagesProviderConfigurationAndBodyLimit(t *testing.T) {
	pages := NewPages()
	if !pages.CanHandle("https://example.com/article") || pages.CanHandle("https://www.youtube.com/watch?v=tidalmill01") {
		t.Fatal("PageProvider.CanHandle misclassified site")
	}
	if isVideoSite("http://\x00") {
		t.Fatal("invalid URL treated as video site")
	}
	if y := newYtDLP(Config{}, process.Runner{}, nil); y.probeTimeout != 2*time.Minute {
		t.Errorf("probeTimeout = %v, want 2m", y.probeTimeout)
	}
	if pages.client.Timeout != 30*time.Second {
		t.Errorf("client.Timeout = %v, want 30s", pages.client.Timeout)
	}
	tr, ok := pages.client.Transport.(*http.Transport)
	if !ok || tr == nil {
		t.Fatalf("client.Transport = %T, want *http.Transport", pages.client.Transport)
	}
	if tr.Proxy == nil || tr.DialContext == nil || !tr.ForceAttemptHTTP2 {
		t.Errorf("transport missing Proxy, DialContext, or ForceAttemptHTTP2: %+v", tr)
	}
	if tr.MaxIdleConns != 100 || tr.MaxIdleConnsPerHost != 20 {
		t.Errorf("idle conns = (%d, %d), want (100, 20)", tr.MaxIdleConns, tr.MaxIdleConnsPerHost)
	}
	if tr.IdleConnTimeout != 90*time.Second || tr.TLSHandshakeTimeout != 10*time.Second || tr.ExpectContinueTimeout != 1*time.Second {
		t.Errorf("timeouts = (%v, %v, %v), want (90s, 10s, 1s)", tr.IdleConnTimeout, tr.TLSHandshakeTimeout, tr.ExpectContinueTimeout)
	}
	body := &paddedHTMLReader{
		prefix: "<html><head><title>T</title></head><body><article><p>Readability body text here.</p></article></body></html>",
		total:  9 << 20,
	}
	pages.client = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       body,
			}, nil
		}),
	}
	if _, err := pages.GetText(t.Context(), "https://example.com/post", PreferredCaptions{}); err != nil {
		t.Fatal(err)
	}
	if body.pos != 8<<20 {
		t.Errorf("GetText read %d bytes, want %d", body.pos, 8<<20)
	}
}

func TestPagesFollowARedirectToAnAddressOnThisMachine(t *testing.T) {
	pages := NewPages()
	via := make([]*http.Request, 9)
	for _, raw := range []string{"http://127.0.0.1/secret", "http://192.168.1.10/"} {
		target, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if err := pages.client.CheckRedirect(&http.Request{URL: target}, via); err != nil {
			t.Errorf("CheckRedirect(%s) after 9 hops = %v, want nil", raw, err)
		}
		if err := pages.client.CheckRedirect(&http.Request{URL: target}, nil); err != nil {
			t.Errorf("CheckRedirect(%s) = %v, want nil", raw, err)
		}
	}
}

func TestPagesStopAfterTenRedirects(t *testing.T) {
	target, err := url.Parse("https://example.com/next")
	if err != nil {
		t.Fatal(err)
	}
	if err := NewPages().client.CheckRedirect(&http.Request{URL: target}, make([]*http.Request, 10)); err == nil {
		t.Error("CheckRedirect after 10 hops = nil, want an error")
	}
}
