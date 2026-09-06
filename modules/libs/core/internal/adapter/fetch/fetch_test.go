package fetch_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/fetch"
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

func address(t *testing.T, written string) domain.WebAddress {
	t.Helper()
	at, err := domain.ParseWebAddress(written)
	if err != nil {
		t.Fatal(err)
	}
	return at
}

// A machine with neither tool has no fetcher at all, which is what says this
// build cannot fetch and takes the run off the palette.
func TestAMachineWithNeitherTool(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if _, err := fetch.New(t.Context(), fetch.Config{}); !errors.Is(err, fetch.ErrNoTool) {
		t.Errorf("a machine holding no tool answered %v", err)
	}
}

// A machine that keeps its tools where nothing may write a path down names
// whatever does know where they are, as a command and what it is started
// through.
func TestAToolNamedAsACommand(t *testing.T) {
	fetcher, err := fetch.New(t.Context(), fetch.Config{Video: fetch.Tool{Command: tool(t, aVideo)}})
	if err != nil {
		t.Fatal(err)
	}

	meta, err := fetcher.Metadata(t.Context(), address(t, "https://youtu.be/dQw4w9WgXcQ"))
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
	if len(meta.Automatic) != 1 || meta.Automatic[0] != "ru" {
		t.Errorf("a machine wrote words in %v", meta.Automatic)
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
	fetcher, err := fetch.New(t.Context(), fetch.Config{Video: fetch.Tool{Command: []string{at}}})
	if err != nil {
		t.Fatal(err)
	}

	_, err = fetcher.Metadata(t.Context(), address(t, "https://youtu.be/dQw4w9WgXcQ"))
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

	fetcher, err := fetch.New(t.Context(), fetch.Config{Video: fetch.Tool{Command: tool(t, aVideo)}})
	if err != nil {
		t.Fatal(err)
	}
	article, err := fetcher.Article(t.Context(), address(t, site.URL+"/entropy"))
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

	fetcher, err := fetch.New(t.Context(), fetch.Config{Video: fetch.Tool{Command: tool(t, aVideo)}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fetcher.Article(t.Context(), address(t, site.URL)); !errors.Is(err, port.ErrNothingFetched) {
		t.Errorf("a page with nothing in it answered %v", err)
	}
}

// A video is not a page: nothing asks a site for the article of something it
// publishes as a video.
func TestAVideoIsNotAPage(t *testing.T) {
	fetcher, err := fetch.New(t.Context(), fetch.Config{Video: fetch.Tool{Command: tool(t, aVideo)}})
	if err != nil {
		t.Fatal(err)
	}
	at := address(t, "https://youtu.be/dQw4w9WgXcQ")
	if _, err := fetcher.Subtitles(t.Context(), domain.WebAddress{URL: "https://example.com/a"},
		"en"); !errors.Is(err, port.ErrNothingFetched) {
		t.Errorf("a page was asked for the words of a video: %v", err)
	}
	if _, err := fetcher.Download(t.Context(), domain.WebAddress{URL: at.URL}, nil); !errors.Is(
		err, port.ErrNothingFetched) {
		t.Errorf("an address with no video at it was copied: %v", err)
	}
}
