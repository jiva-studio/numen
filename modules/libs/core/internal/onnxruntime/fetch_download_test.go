package onnxruntime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// trustServer lets the download client talk to a test server's certificate.
func trustServer(t *testing.T, server *httptest.Server) {
	t.Helper()
	previous := fetching.Transport
	fetching.Transport = server.Client().Transport
	t.Cleanup(func() { fetching.Transport = previous })
}

func newTLSServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	trustServer(t, server)
	return server
}

func downloading(dir string) Settings {
	return Settings{Dir: dir, Section: "indexing.recognition", ShouldDownload: true}
}

func assertNothingKept(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		t.Errorf("%s was left in the folder", entry.Name())
	}
}

// A fetched file is kept whole under its cache name, with no partial file
// beside it, and the progress is reported from zero to the length.
func TestFetchDownloadsTheFileAndReportsProgress(t *testing.T) {
	body := bytes.Repeat([]byte("onnx"), 1000)
	server := newTLSServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Write(body)
	})
	dir := t.TempDir()
	address := server.URL + "/models/encoder.onnx"
	type report struct {
		what        string
		done, total int64
	}
	var reports []report
	s := downloading(dir)
	s.Fetching = func(what string, done, total int64) { reports = append(reports, report{what, done, total}) }

	got, err := Fetch(t.Context(), s, address)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, GetCacheName(address)); got != want {
		t.Fatalf("Fetch = %q, want %q", got, want)
	}
	kept, err := os.ReadFile(got)
	if err != nil || !bytes.Equal(kept, body) {
		t.Fatalf("kept %d bytes, %v", len(kept), err)
	}
	if _, err := os.Stat(got + ".part"); err == nil {
		t.Error("the partial file was left")
	}
	if len(reports) < 2 || reports[0] != (report{"encoder.onnx", 0, 0}) {
		t.Fatalf("reports = %v, want a start at zero first", reports)
	}
	last := reports[len(reports)-1]
	if last.what != "encoder.onnx" || last.done <= 0 || last.done > int64(len(body)) || last.total != int64(len(body)) {
		t.Fatalf("last report = %v", last)
	}
	again, err := Fetch(t.Context(), Settings{Dir: dir}, address)
	if err != nil || again != got {
		t.Fatalf("a kept file is fetched again: %q, %v", again, err)
	}
}

// A folder that does not exist yet is made.
func TestFetchMakesTheFolderItKeepsFilesIn(t *testing.T) {
	server := newTLSServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("x")) })
	dir := filepath.Join(t.TempDir(), "a", "b")
	got, err := Fetch(t.Context(), downloading(dir), server.URL+"/m.onnx")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(got) != dir {
		t.Fatalf("kept at %q", got)
	}
}

// A folder that cannot be made is an error, and nothing is requested.
func TestFetchFailsWhenTheFolderCannotBeMade(t *testing.T) {
	var asked atomic.Bool
	server := newTLSServer(t, func(w http.ResponseWriter, r *http.Request) { asked.Store(true) })
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Fetch(t.Context(), downloading(filepath.Join(blocker, "models")), server.URL+"/m.onnx")
	if err == nil || got != "" {
		t.Fatalf("Fetch = %q, %v; want no path and an error", got, err)
	}
	if asked.Load() {
		t.Error("the host was asked")
	}
}

// Only an https address is downloaded.
func TestFetchRefusesAnAddressThatIsNotHTTPS(t *testing.T) {
	var asked atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { asked.Store(true) }))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	_, err := Fetch(t.Context(), downloading(dir), server.URL+"/m.onnx")
	if err == nil || !strings.Contains(err.Error(), "not an https address") {
		t.Fatalf("err = %v", err)
	}
	if asked.Load() {
		t.Error("the host was asked")
	}
	assertNothingKept(t, dir)
}

func TestIsAddressIsHTTPSOnly(t *testing.T) {
	for address, want := range map[string]bool{
		"https://host/m.onnx": true,
		"http://host/m.onnx":  false,
		"/models/m.onnx":      false,
		"":                    false,
	} {
		if got := IsAddress(address); got != want {
			t.Errorf("IsAddress(%q) = %v", address, got)
		}
	}
}

// A host that answers with anything but 200 gives no file.
func TestFetchFailsOnAnAnswerThatIsNotOK(t *testing.T) {
	server := newTLSServer(t, func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	dir := t.TempDir()
	_, err := Fetch(t.Context(), downloading(dir), server.URL+"/m.onnx")
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("err = %v", err)
	}
	assertNothingKept(t, dir)
}

// A host that cannot be reached gives no file, and the error names the address.
func TestFetchFailsWhenTheHostIsUnreachable(t *testing.T) {
	server := newTLSServer(t, func(w http.ResponseWriter, r *http.Request) {})
	address := server.URL + "/m.onnx"
	server.Close()
	dir := t.TempDir()
	_, err := Fetch(t.Context(), downloading(dir), address)
	if err == nil || !strings.Contains(err.Error(), address) {
		t.Fatalf("err = %v", err)
	}
	assertNothingKept(t, dir)
}

// An address that is not a URL gives no request.
func TestDownloadFailsOnAnAddressThatIsNotAURL(t *testing.T) {
	dir := t.TempDir()
	err := download(t.Context(), downloading(dir), "https://\x7f/m.onnx", filepath.Join(dir, "m.onnx"))
	if err == nil {
		t.Fatal("no error")
	}
	assertNothingKept(t, dir)
}

// A body that ends before its announced length leaves no file, finished or
// not, and the error keeps its cause.
func TestFetchLeavesNothingWhenTheBodyEndsEarly(t *testing.T) {
	server := newTLSServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.Write([]byte("short"))
	})
	dir := t.TempDir()
	_, err := Fetch(t.Context(), downloading(dir), server.URL+"/m.onnx")
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want it to wrap %v", err, io.ErrUnexpectedEOF)
	}
	assertNothingKept(t, dir)
}

// A file that cannot be created is an error.
func TestDownloadFailsWhenThePartialFileCannotBeCreated(t *testing.T) {
	server := newTLSServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("x")) })
	missing := filepath.Join(t.TempDir(), "missing", "m.onnx")
	if err := download(t.Context(), downloading(filepath.Dir(missing)), server.URL+"/m.onnx", missing); err == nil {
		t.Fatal("no error")
	}
}

// A file that cannot be put in its place is an error.
func TestDownloadFailsWhenTheFileCannotBePutInItsPlace(t *testing.T) {
	server := newTLSServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("x")) })
	at := filepath.Join(t.TempDir(), "m.onnx")
	if err := os.MkdirAll(filepath.Join(at, "inside"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := download(t.Context(), downloading(filepath.Dir(at)), server.URL+"/m.onnx", at); err == nil {
		t.Fatal("no error")
	}
}

// A cancelled context stops the download.
func TestDownloadStopsWhenItsContextIsCancelled(t *testing.T) {
	server := newTLSServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("x")) })
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	dir := t.TempDir()
	if _, err := Fetch(ctx, downloading(dir), server.URL+"/m.onnx"); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	assertNothingKept(t, dir)
}

// redirecting answers the first `hops` requests with a redirect to itself and
// the next with a file.
func redirecting(hops int64) http.HandlerFunc {
	var served atomic.Int64
	return func(w http.ResponseWriter, r *http.Request) {
		if n := served.Add(1); n <= hops {
			http.Redirect(w, r, fmt.Sprintf("/hop/%d", n), http.StatusFound)
			return
		}
		w.Write([]byte("arrived"))
	}
}

// Nine redirects are followed, and a tenth is refused.
func TestFetchFollowsNineRedirectsAndRefusesTen(t *testing.T) {
	nine := newTLSServer(t, redirecting(9))
	if _, err := Fetch(t.Context(), downloading(t.TempDir()), nine.URL+"/m.onnx"); err != nil {
		t.Fatalf("nine redirects: %v", err)
	}
	ten := newTLSServer(t, redirecting(10))
	dir := t.TempDir()
	_, err := Fetch(t.Context(), downloading(dir), ten.URL+"/m.onnx")
	if err == nil || !strings.Contains(err.Error(), ten.URL+"/m.onnx redirects ten times") {
		t.Fatalf("ten redirects: %v", err)
	}
	assertNothingKept(t, dir)
}

// A redirect to another https host is followed.
func TestFetchFollowsARedirectToAnotherHTTPSHost(t *testing.T) {
	target := newTLSServer(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("moved")) })
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/there.onnx", http.StatusFound)
	}))
	t.Cleanup(origin.Close)
	got, err := Fetch(t.Context(), downloading(t.TempDir()), origin.URL+"/m.onnx")
	if err != nil {
		t.Fatal(err)
	}
	if kept, _ := os.ReadFile(got); string(kept) != "moved" {
		t.Fatalf("kept %q", kept)
	}
}

// A redirect to a plain http address is refused, and the error names both ends.
func TestFetchRefusesARedirectToPlainHTTP(t *testing.T) {
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("plain")) }))
	t.Cleanup(plain.Close)
	origin := newTLSServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+"/m.onnx", http.StatusFound)
	})
	dir := t.TempDir()
	_, err := Fetch(t.Context(), downloading(dir), origin.URL+"/m.onnx")
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("%s/m.onnx redirects to %s/m.onnx", origin.URL, plain.URL)) {
		t.Fatalf("err = %v", err)
	}
	assertNothingKept(t, dir)
}

// failingWriter reports two bytes written and fails.
type failingWriter struct{ err error }

func (w failingWriter) Write(raw []byte) (int, error) { return 2, w.err }

// The counting writer passes every write on and returns what the writer under it
// returned.
func TestCountingPassesWritesOn(t *testing.T) {
	var out bytes.Buffer
	c := &counting{to: &out, total: 9, say: func(int64, int64) {}}
	n, err := c.Write([]byte("abcd"))
	if n != 4 || err != nil || out.String() != "abcd" {
		t.Fatalf("Write = %d, %v, %q", n, err, out.String())
	}
	broken := errors.New("disk full")
	c = &counting{to: failingWriter{broken}, say: func(int64, int64) {}}
	if n, err := c.Write([]byte("abcd")); n != 2 || err != broken {
		t.Fatalf("Write = %d, %v", n, err)
	}
}

// The counting writer reports its running total at most once a second.
func TestCountingReportsOnceASecond(t *testing.T) {
	type report struct{ done, total int64 }
	var reports []report
	c := &counting{to: io.Discard, total: 9, say: func(done, total int64) { reports = append(reports, report{done, total}) }}

	c.Write([]byte("abc"))
	c.Write([]byte("d"))
	if len(reports) != 1 || reports[0] != (report{3, 9}) {
		t.Fatalf("within a second: reports = %v", reports)
	}
	c.last = time.Now().Add(-2 * time.Second)
	c.Write([]byte("ef"))
	if len(reports) != 2 || reports[1] != (report{6, 9}) {
		t.Fatalf("after a second: reports = %v", reports)
	}
	c.Write([]byte("g"))
	if len(reports) != 2 {
		t.Fatalf("a report followed the last at once: %v", reports)
	}
}
