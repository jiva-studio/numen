package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func sumOf(body string) string {
	digest := sha256.Sum256([]byte(body))
	return hex.EncodeToString(digest[:])
}

// serve answers every request with body over https, and trusts its certificate.
func serve(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	previous := client.Transport
	client.Transport = srv.Client().Transport
	t.Cleanup(func() { client.Transport = previous })
	return srv
}

func request(t *testing.T, address, sum string) Request {
	t.Helper()
	t.Setenv(EnvOffline, "")
	t.Setenv(EnvQuota, "")
	dir := filepath.Join(t.TempDir(), "models")
	t.Setenv(EnvDir, dir)
	return Request{Dir: dir, Section: "indexing.test", Address: address, Sum: sum, ShouldDownload: true}
}

func TestFetchKeepsAPinnedFileUnderItsSum(t *testing.T) {
	srv := serve(t, "weights")
	var told bool
	r := request(t, srv.URL+"/m/model.onnx", sumOf("weights"))
	r.Progress = func(what string, _, _ int64) { told = told || what == "model.onnx" }

	at, err := Fetch(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(r.Dir, sumOf("weights")[:12]+"-model.onnx"); at != want {
		t.Fatalf("Fetch() = %q, want %q", at, want)
	}
	if got, _ := os.ReadFile(at); string(got) != "weights" {
		t.Fatalf("file holds %q", got)
	}
	if !told {
		t.Fatal("progress was not reported")
	}
	if _, err := os.Stat(at + ".part"); !os.IsNotExist(err) {
		t.Fatalf("part file remains: %v", err)
	}
}

func TestFetchRefusesAFileWithAnotherSum(t *testing.T) {
	srv := serve(t, "tampered")
	r := request(t, srv.URL+"/model.onnx", sumOf("weights"))

	_, err := Fetch(context.Background(), r)
	if err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("Fetch() error = %v", err)
	}
	entries, _ := os.ReadDir(r.Dir)
	if len(entries) != 0 {
		t.Fatalf("the folder holds %d files after a refused download", len(entries))
	}
}

func TestFetchTakesAnUnpinnedFileAsNamed(t *testing.T) {
	srv := serve(t, "anything")
	r := request(t, srv.URL+"/model.onnx", "")

	at, err := Fetch(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if at != filepath.Join(r.Dir, GetCacheName(r.Address, "")) {
		t.Fatalf("Fetch() = %q", at)
	}
}

func TestFetchReturnsAFileThatIsPresentWithoutTheNetwork(t *testing.T) {
	r := request(t, "https://example.invalid/model.onnx", sumOf("weights"))
	r.ShouldDownload = false
	at := filepath.Join(r.Dir, GetCacheName(r.Address, r.Sum))
	if err := os.MkdirAll(r.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvOffline, "1")

	got, err := Fetch(context.Background(), r)
	if err != nil || got != at {
		t.Fatalf("Fetch() = %q, %v", got, err)
	}
}

func TestFetchNamesTheSettingWhenDownloadsAreOff(t *testing.T) {
	r := request(t, "https://example.invalid/model.onnx", "")
	r.ShouldDownload = false
	_, err := Fetch(context.Background(), r)
	if err == nil || !strings.Contains(err.Error(), "indexing.test.download") {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestFetchRefusesWhenOffline(t *testing.T) {
	srv := serve(t, "weights")
	r := request(t, srv.URL+"/model.onnx", "")
	t.Setenv(EnvOffline, "true")
	_, err := Fetch(context.Background(), r)
	if err == nil || !strings.Contains(err.Error(), EnvOffline) {
		t.Fatalf("Fetch() error = %v", err)
	}
	if _, err := os.Stat(r.Dir); !os.IsNotExist(err) {
		t.Fatalf("the folder was made while offline: %v", err)
	}
}

func TestFetchRefusesAnAddressThatIsNotHTTPS(t *testing.T) {
	r := request(t, "http://example.invalid/model.onnx", "")
	if _, err := Fetch(context.Background(), r); err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestFetchRefusesADownloadPastTheQuotaByItsLength(t *testing.T) {
	srv := serve(t, "0123456789")
	r := request(t, srv.URL+"/model.onnx", "")
	t.Setenv(EnvQuota, "4")
	if _, err := Fetch(context.Background(), r); err == nil || !strings.Contains(err.Error(), EnvQuota) {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestFetchStopsAStreamPastTheQuota(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		f := w.(http.Flusher)
		_, _ = w.Write([]byte("0123456789"))
		f.Flush()
		_, _ = w.Write([]byte("0123456789"))
	}))
	t.Cleanup(srv.Close)
	previous := client.Transport
	client.Transport = srv.Client().Transport
	t.Cleanup(func() { client.Transport = previous })
	r := request(t, srv.URL+"/model.onnx", "")
	t.Setenv(EnvQuota, "15")

	_, err := Fetch(context.Background(), r)
	if err == nil || !strings.Contains(err.Error(), EnvQuota) {
		t.Fatalf("Fetch() error = %v", err)
	}
	if entries, _ := os.ReadDir(r.Dir); len(entries) != 0 {
		t.Fatalf("the folder holds %d files after a refused download", len(entries))
	}
}

func TestFetchReportsAnErrorStatus(t *testing.T) {
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	previous := client.Transport
	client.Transport = srv.Client().Transport
	t.Cleanup(func() { client.Transport = previous })
	r := request(t, srv.URL+"/model.onnx", "")
	if _, err := Fetch(context.Background(), r); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestClientFollowsRedirectsOnlyToHTTPS(t *testing.T) {
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("x")) }))
	t.Cleanup(plain.Close)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL, http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	previous := client.Transport
	client.Transport = srv.Client().Transport
	t.Cleanup(func() { client.Transport = previous })
	r := request(t, srv.URL+"/model.onnx", "")
	if _, err := Fetch(context.Background(), r); err == nil || !strings.Contains(err.Error(), "redirects") {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestClientStopsAfterTenRedirects(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL+"/again", http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	previous := client.Transport
	client.Transport = srv.Client().Transport
	t.Cleanup(func() { client.Transport = previous })
	r := request(t, srv.URL+"/model.onnx", "")
	if _, err := Fetch(context.Background(), r); err == nil || !strings.Contains(err.Error(), "ten times") {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestVerifyFile(t *testing.T) {
	name := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(name, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyFile(name, strings.ToUpper(sumOf("weights"))); err != nil {
		t.Fatalf("matching sum: %v", err)
	}
	if err := VerifyFile(name, sumOf("other")); err == nil {
		t.Fatal("another sum passed")
	}
	if err := VerifyFile(filepath.Join(t.TempDir(), "missing"), sumOf("x")); err == nil {
		t.Fatal("a missing file passed")
	}
}

func TestGetDownloadTimeout(t *testing.T) {
	if getDownloadTimeout(0) <= getDownloadTimeout(1<<20) {
		t.Fatal("an unknown size must have longer than a megabyte")
	}
	if getDownloadTimeout(1) < headerTimeout {
		t.Fatal("every file has at least the header timeout")
	}
}

func TestGetCacheName(t *testing.T) {
	if got := GetCacheName("https://h/a/model.onnx", "abcdef0123456789"); got != "abcdef012345-model.onnx" {
		t.Fatalf("pinned name = %q", got)
	}
	if got := GetCacheName("https://h/a/model.onnx", "ab"); got != "ab-model.onnx" {
		t.Fatalf("short sum name = %q", got)
	}
	one, other := GetCacheName("https://h/a/model.onnx", ""), GetCacheName("https://h/b/model.onnx", "")
	if one == other || !strings.HasSuffix(one, "-model.onnx") {
		t.Fatalf("names %q and %q", one, other)
	}
	if got := GetCacheName("", ""); !strings.HasSuffix(got, "-model.onnx") {
		t.Fatalf("empty base name = %q", got)
	}
}

func TestIsAddress(t *testing.T) {
	if !IsAddress("https://h/x") || IsAddress("http://h/x") || IsAddress("model.onnx") {
		t.Fatal("IsAddress accepts only https")
	}
}

func TestFetchRefusesAPresentPinnedFileWithAnotherSum(t *testing.T) {
	r := request(t, "https://example.invalid/model.onnx", sumOf("weights"))
	r.ShouldDownload = false
	if err := os.MkdirAll(r.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	at := filepath.Join(r.Dir, GetCacheName(r.Address, r.Sum))
	if err := os.WriteFile(at, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Fetch(context.Background(), r); err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestFetchCountsTheSharedFolderAgainstTheQuota(t *testing.T) {
	srv := serve(t, "0123456789")
	r := request(t, srv.URL+"/model.onnx", "")
	if err := os.MkdirAll(r.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.Dir, "other"), make([]byte, 8), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvQuota, "12")
	if _, err := Fetch(context.Background(), r); err == nil || !strings.Contains(err.Error(), EnvQuota) {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestFetchKeepsNoTemporaryFileAfterSuccess(t *testing.T) {
	srv := serve(t, "weights")
	r := request(t, srv.URL+"/model.onnx", "")
	if _, err := Fetch(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(r.Dir)
	if len(entries) != 1 || strings.Contains(entries[0].Name(), ".part") {
		t.Fatalf("the folder holds %v", entries)
	}
}

func TestFetchReplacesAPresentPinnedFileWithAnotherSumWhenDownloadsAreAllowed(t *testing.T) {
	srv := serve(t, "weights")
	r := request(t, srv.URL+"/model.onnx", sumOf("weights"))
	if err := os.MkdirAll(r.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	at := filepath.Join(r.Dir, GetCacheName(r.Address, r.Sum))
	if err := os.WriteFile(at, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Fetch(context.Background(), r)
	if err != nil || got != at {
		t.Fatalf("Fetch() = %q, %v", got, err)
	}
	if body, _ := os.ReadFile(at); string(body) != "weights" {
		t.Fatalf("file holds %q", body)
	}
}

func TestFetchRefusesAPresentPinnedFileWithAnotherSumWhileOffline(t *testing.T) {
	r := request(t, "https://example.invalid/model.onnx", sumOf("weights"))
	if err := os.MkdirAll(r.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	at := filepath.Join(r.Dir, GetCacheName(r.Address, r.Sum))
	if err := os.WriteFile(at, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvOffline, "1")
	if _, err := Fetch(context.Background(), r); err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestFetchNamesTheFileWithoutTheQueryOfItsAddress(t *testing.T) {
	srv := serve(t, "weights")
	r := request(t, srv.URL+"/model.onnx?download=true", "")
	at, err := Fetch(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(at, "-model.onnx") {
		t.Fatalf("Fetch() = %q", at)
	}
}

func TestFetchReportsAFolderItCannotCreate(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	r := request(t, "https://example.invalid/model.onnx", "")
	r.Dir = filepath.Join(blocker, "models")
	if _, err := Fetch(context.Background(), r); err == nil {
		t.Fatal("a download was made under a file")
	}
}

func TestGetQuotaRootFallsBackToTheRequestFolder(t *testing.T) {
	t.Setenv(EnvDir, "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("AppData", "")
	if runtime.GOOS != "linux" {
		t.Skip("only linux reads no home folder from an empty environment")
	}
	if got := getQuotaRoot(Request{Dir: "/srv/here"}); got != "/srv/here" {
		t.Fatalf("getQuotaRoot() = %q", got)
	}
}
