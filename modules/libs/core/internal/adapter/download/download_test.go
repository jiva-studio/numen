package download_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/download"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A tool this machine holds, which answers what it is told to answer. Nothing
// here reaches a network: what a run of the real tool says is recorded, and
// this hands that back.
func tool(t *testing.T, says string) []string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the tool is a shell script")
	}
	at := filepath.Join(t.TempDir(), "yt-dlp")
	written := "#!/bin/sh\ncat <<'ANSWER'\n" + says + "\nANSWER\n"
	if err := os.WriteFile(at, []byte(written), 0o700); err != nil {
		t.Fatal(err)
	}
	return []string{at}
}

const aVideo = `{"title":"Entropy explained","duration":83.5,` +
	`"subtitles":{"en":[{}]},"automatic_captions":{"ru":[{}]}}`

func address(t *testing.T, written string) domain.URL {
	t.Helper()
	at, err := domain.ParseURL(written)
	if err == nil {
		return at
	}
	// A site standing on this machine is one no note may point at, and it is
	// where a test puts one. The downloader is handed the address, and reading one
	// is the vault's own step.
	if !strings.HasPrefix(written, "http://127.0.0.1:") {
		t.Fatal(err)
	}
	return domain.URL(written)
}

// A machine with neither tool still fetches a page: an address that is not a
// video is an ordinary request, and the prose is found in this process.
func TestAMachineWithNeitherToolFetchesAPage(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><html><head><title>Entropy</title></head><body>
			<article><h1>Entropy</h1><p>A measure of how many ways the parts of a thing
			can be arranged, and it grows.</p></article></body></html>`))
	}))
	defer site.Close()

	downloader, err := download.New(t.Context(), download.Config{})
	if err != nil {
		t.Fatal(err)
	}
	article, err := downloader.Text(t.Context(), address(t, site.URL+"/entropy"), port.PreferredCaptions{})
	if err != nil {
		t.Fatalf("a page went unfetched on a machine holding no tool: %v", err)
	}
	if !strings.Contains(article.Prose, "arranged") {
		t.Errorf("the prose reads %q", article.Prose)
	}
}

// A video on such a machine says what is missing. yt-dlp is what reaches one,
// on the sites it knows, and nothing here reaches one without it.
func TestAVideoOnAMachineWithNeitherTool(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	downloader, err := download.New(t.Context(), download.Config{})
	if err != nil {
		t.Fatal(err)
	}
	at := address(t, "https://youtu.be/dQw4w9WgXcQ")
	if _, err := downloader.Metadata(t.Context(), at); !errors.Is(err, download.ErrNoTool) {
		t.Errorf("a video was looked at without the tool that reaches one: %v", err)
	}
	if _, err := downloader.Download(t.Context(), at, io.Discard); !errors.Is(err, download.ErrNoTool) {
		t.Errorf("a video was copied without the tool that reaches one: %v", err)
	}
}

// A machine that keeps its tools where nothing may write a path down names
// whatever does know where they are, as a command and what it is started
// through.
func TestAToolNamedAsACommand(t *testing.T) {
	downloader, err := download.New(t.Context(), download.Config{Video: download.Tool{Command: tool(t, aVideo)}})
	if err != nil {
		t.Fatal(err)
	}

	meta, err := downloader.Metadata(t.Context(), address(t, "https://youtu.be/dQw4w9WgXcQ"))
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "Entropy explained" {
		t.Errorf("it is called %q", meta.Title)
	}
	if meta.Length != 83_500 {
		t.Errorf("it runs %d ms", meta.Length)
	}
	// What a person published and what a machine wrote are two lists: they are
	// worth different amounts, and only one of them is asked for.
	if len(meta.Captions) != 1 || meta.Captions[0] != "en" {
		t.Errorf("a person published words in %v", meta.Captions)
	}
	if len(meta.IsAutomatic) != 1 || meta.IsAutomatic[0] != "ru" {
		t.Errorf("a machine wrote words in %v", meta.IsAutomatic)
	}
}

// What the tool said when it refused is what the person is shown: it knows why,
// and nothing here is going to say it better.
func TestWhatTheToolSaidWhenItRefused(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the tool is a shell script")
	}
	at := filepath.Join(t.TempDir(), "yt-dlp")
	written := "#!/bin/sh\n" +
		"echo 'ERROR: Sign in to confirm you are not a bot' 1>&2\nexit 1\n"
	if err := os.WriteFile(at, []byte(written), 0o700); err != nil {
		t.Fatal(err)
	}
	downloader, err := download.New(t.Context(), download.Config{Video: download.Tool{Command: []string{at}}})
	if err != nil {
		t.Fatal(err)
	}

	_, err = downloader.Metadata(t.Context(), address(t, "https://youtu.be/dQw4w9WgXcQ"))
	if err == nil || !strings.Contains(err.Error(), "not a bot") {
		t.Errorf("the refusal reads %v", err)
	}
}

// A page is fetched as a browser fetches one, and what comes back is the
// article without the furniture around it.
func TestThePageWithoutTheFurnitureAroundIt(t *testing.T) {
	held := `<!doctype html><html><head><title>Entropy — a page</title></head><body>
		<nav>Home About Contact</nav>
		<article><h1>Entropy</h1>
		<p>A measure of how many ways the parts of a thing can be arranged.</p>
		<p>It grows, and the arrow of time is drawn from that and nothing else.</p>
		</article><footer>Copyright</footer></body></html>`
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(held))
	}))
	defer site.Close()

	downloader, err := download.New(t.Context(), download.Config{Video: download.Tool{Command: tool(t, aVideo)}})
	if err != nil {
		t.Fatal(err)
	}
	article, err := downloader.Text(t.Context(), address(t, site.URL+"/entropy"), port.PreferredCaptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(article.Prose, "arrow of time") {
		t.Errorf("the prose reads %q", article.Prose)
	}
	if strings.Contains(article.Prose, "Copyright") {
		t.Errorf("the furniture came with it: %q", article.Prose)
	}
	if article.Title == "" {
		t.Error("the page is called nothing")
	}
}

// A page that answered with no article is an answer and not a failure: nothing
// was published there to keep.
func TestAPageWithNoArticleInIt(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!doctype html><html><body></body></html>"))
	}))
	defer site.Close()

	downloader, err := download.New(t.Context(), download.Config{Video: download.Tool{Command: tool(t, aVideo)}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := downloader.Text(t.Context(), address(t, site.URL), port.PreferredCaptions{}); !errors.Is(err, port.ErrNothingDownloaded) {
		t.Errorf("a page with nothing in it answered %v", err)
	}
}

// A video is not a page: nothing asks a site for the article of something it
// publishes as a video.
func TestAVideoIsNotAPage(t *testing.T) {
	downloader, err := download.New(t.Context(), download.Config{Video: download.Tool{Command: tool(t, aVideo)}})
	if err != nil {
		t.Fatal(err)
	}
	at := address(t, "https://youtu.be/dQw4w9WgXcQ")
	if _, err := downloader.Download(t.Context(), at, nil); !errors.Is(
		err, port.ErrNothingDownloaded) {
		t.Errorf("an address with no video at it was copied: %v", err)
	}
}

// ffmpeg reaches no address of its own. It brings what another tool took to the
// container a transcriber opens, so a machine holding it alone reaches a video
// no better than a machine holding nothing.
func TestAMachineWithFfmpegAndNothingThatReachesAVideo(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	downloader, err := download.New(t.Context(), download.Config{Sound: download.Tool{Command: tool(t, "")}})
	if err != nil {
		t.Fatal(err)
	}
	at := address(t, "https://youtu.be/dQw4w9WgXcQ")
	if _, err := downloader.Metadata(t.Context(), at); !errors.Is(err, download.ErrNoTool) {
		t.Errorf("a machine holding only ffmpeg looked at a video: %v", err)
	}
}

// Which provider answers is which one supports the address, and the first that
// does is the one asked. What a fetch is claimed by says which of them it was,
// so a text kept beyond the run is claimed again by what made it.
func TestWhichProviderSupportsAnAddress(t *testing.T) {
	downloader, err := download.New(t.Context(), download.Config{Video: download.Tool{Command: tool(t, aVideo)}})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range []struct {
		written string
		want    string
	}{
		{"https://youtu.be/dQw4w9WgXcQ", "yt-dlp"},
		{"https://example.com/entropy", "go-readability"},
	} {
		if got := downloader.GetDownloadModel(address(t, one.written)).Tool; got != one.want {
			t.Errorf("%s is fetched by %q, want %q", one.written, got, one.want)
		}
	}
}

// A tool that finds this machine's certificates by an environment variable is
// told which. A machine that keeps them where the tool does not look — which is
// every machine whose store is built afresh — has nowhere else to say so, and
// what comes back is a refusal about certificates and not about the video.
func TestAToolIsStartedWithTheEnvironmentTheSettingsName(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the tool is a shell script")
	}
	at := filepath.Join(t.TempDir(), "yt-dlp")
	// The tool answers with what it was started with, so what reached it is
	// what comes back.
	written := "#!/bin/sh\nprintf '{\"title\":\"%s\"}\\n' \"$SSL_CERT_FILE\"\n"
	if err := os.WriteFile(at, []byte(written), 0o700); err != nil {
		t.Fatal(err)
	}
	downloader, err := download.New(t.Context(), download.Config{Video: download.Tool{
		Command:     []string{at},
		Environment: map[string]string{"SSL_CERT_FILE": "/etc/ssl/certs/ca-certificates.crt"},
	}})
	if err != nil {
		t.Fatal(err)
	}

	meta, err := downloader.Metadata(t.Context(), address(t, "https://youtu.be/dQw4w9WgXcQ"))
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "/etc/ssl/certs/ca-certificates.crt" {
		t.Errorf("the tool was started with SSL_CERT_FILE=%q", meta.Title)
	}
}

// A tool the settings name no environment for is started with this process's
// own: a machine that needs nothing said keeps what it already had.
func TestAToolNamedNoEnvironmentKeepsThisProcessesOwn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the tool is a shell script")
	}
	t.Setenv("NUMEN_TEST_MARK", "what this process was started with")
	at := filepath.Join(t.TempDir(), "yt-dlp")
	written := "#!/bin/sh\nprintf '{\"title\":\"%s\"}\\n' \"$NUMEN_TEST_MARK\"\n"
	if err := os.WriteFile(at, []byte(written), 0o700); err != nil {
		t.Fatal(err)
	}
	downloader, err := download.New(t.Context(), download.Config{
		Video: download.Tool{Command: []string{at}},
	})
	if err != nil {
		t.Fatal(err)
	}

	meta, err := downloader.Metadata(t.Context(), address(t, "https://youtu.be/dQw4w9WgXcQ"))
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "what this process was started with" {
		t.Errorf("the tool was started with NUMEN_TEST_MARK=%q", meta.Title)
	}
}
