//go:build unix

package bind

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// startIn starts the core under dir and stops it when the test ends. The
// environment the start rewrites is put back with it.
func startIn(t *testing.T, dir string) int {
	t.Helper()
	keepEnvironment(t)
	port, err := Start(dir)
	if err != nil {
		t.Fatalf("starting: %v", err)
	}
	t.Cleanup(func() { _ = Stop() })
	return port
}

func keepEnvironment(t *testing.T) {
	t.Helper()
	for _, named := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME"} {
		t.Setenv(named, os.Getenv(named))
	}
}

// captureStderr collects what the process writes on its error stream until the
// returned function is called, which answers with it.
func captureStderr(t *testing.T) func() string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stderr
	os.Stderr = writer
	var collected bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&collected, reader)
		close(done)
	}()
	return func() string {
		os.Stderr = previous
		_ = writer.Close()
		<-done
		_ = reader.Close()
		return collected.String()
	}
}

func setUmask(t *testing.T, mask int) {
	t.Helper()
	previous := syscall.Umask(mask)
	t.Cleanup(func() { syscall.Umask(previous) })
}

func TestStartWhileStandingAnswersThePortItHas(t *testing.T) {
	first := startIn(t, t.TempDir())
	second, err := Start(t.TempDir())
	if err != nil {
		t.Fatalf("starting again: %v", err)
	}
	if second != first {
		t.Errorf("started again on %d, want the port %d it already has", second, first)
	}
	if got := Port(); got != first {
		t.Errorf("Port() = %d, want %d", got, first)
	}
}

func TestStartPointsEveryFolderLookupAtTheFolderItWasGiven(t *testing.T) {
	dir := t.TempDir()
	startIn(t, dir)
	for _, named := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME"} {
		if got := os.Getenv(named); got != dir {
			t.Errorf("%s = %q, want %q", named, got, dir)
		}
	}
}

func TestStartAnswersTheErrorOfAFolderTheEnvironmentRefuses(t *testing.T) {
	keepEnvironment(t)
	port, err := Start("a\x00folder")
	var refused *os.SyscallError
	if !errors.As(err, &refused) || refused.Syscall != "setenv" || port != 0 {
		t.Fatalf("Start answered %d, %v, want 0 and the error of setting the environment", port, err)
	}
	if Port() != 0 {
		t.Errorf("a start that failed listens on %d", Port())
	}
}

func TestStartAnswersTheErrorOfAFolderItCannotMakeAVaultIn(t *testing.T) {
	keepEnvironment(t)
	file := filepath.Join(t.TempDir(), "a-file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	port, err := Start(file)
	var failed *fs.PathError
	if !errors.As(err, &failed) || failed.Op != "mkdir" || port != 0 {
		_ = Stop()
		t.Fatalf("Start answered %d, %v for a folder that is a file, want 0 and the error of making the vault folder", port, err)
	}
}

func TestStartMakesTheVaultFolderReadableByEveryoneAndSeedsItReadable(t *testing.T) {
	setUmask(t, 0)
	dir := t.TempDir()
	startIn(t, dir)
	root := filepath.Join(dir, "vault")
	info, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Errorf("the vault folder is %v, want 0755", got)
	}
	seeded, err := os.Stat(filepath.Join(root, Seeded))
	if err != nil {
		t.Fatal(err)
	}
	if got := seeded.Mode().Perm(); got != 0o644 {
		t.Errorf("the seeded note is %v, want 0644", got)
	}
}

func TestStartAnswersTheErrorOfAVaultFolderItCannotRead(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a superuser reads any folder")
	}
	keepEnvironment(t)
	dir := t.TempDir()
	root := filepath.Join(dir, "vault")
	if err := os.Mkdir(root, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o700) })
	port, err := Start(dir)
	var failed *fs.PathError
	if !errors.As(err, &failed) || failed.Op != "open" || !errors.Is(err, fs.ErrPermission) || port != 0 {
		_ = Stop()
		t.Fatalf("Start answered %d, %v for a vault folder it cannot read, want 0 and the error of reading it", port, err)
	}
	checkNoIndexOpened(t, dir)
}

func TestStartAnswersTheErrorOfAVaultListItCannotRead(t *testing.T) {
	keepEnvironment(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vaults.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	port, err := Start(dir)
	var failed *json.SyntaxError
	if !errors.As(err, &failed) || port != 0 {
		_ = Stop()
		t.Fatalf("Start answered %d, %v for a vault list that is not JSON, want 0 and the error of reading it", port, err)
	}
	checkNoIndexOpened(t, dir)
	if Port() != 0 {
		t.Errorf("a start that failed listens on %d", Port())
	}
}

func TestStartAnswersTheErrorOfAnIndexItCannotOpen(t *testing.T) {
	keepEnvironment(t)
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "index.db"), 0o755); err != nil {
		t.Fatal(err)
	}
	port, err := Start(dir)
	if err == nil || !strings.Contains(err.Error(), "unable to open database file") || port != 0 {
		_ = Stop()
		t.Fatalf("Start answered %d, %v for an index that is a folder, want 0 and the error of opening it", port, err)
	}
}

func TestStartSaysWhenTheThemesFolderCannotBeMadeAndStillStarts(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "themes"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	read := captureStderr(t)
	startIn(t, dir)
	if said := read(); !strings.Contains(said, "themes:") {
		t.Errorf("nothing said the themes folder could not be made: %q", said)
	}
}

func TestStartSaysNothingOfThemesWhenTheirFolderIsThere(t *testing.T) {
	read := captureStderr(t)
	startIn(t, t.TempDir())
	if said := read(); strings.Contains(said, "themes:") {
		t.Errorf("themes were complained of: %q", said)
	}
}

func TestStartGivesTheWindowTheInstallationsThemes(t *testing.T) {
	startIn(t, t.TempDir())
	mu.Lock()
	defer mu.Unlock()
	if running.opened.API.Themes == nil {
		t.Error("the installation holds no themes")
	}
}

func TestStopWithNothingRunningDoesNothing(t *testing.T) {
	if err := Stop(); err != nil {
		t.Fatalf("Stop() = %v", err)
	}
}

func TestStopEndsTheServerAndFreesTheCoreToStartAgain(t *testing.T) {
	read := captureStderr(t)
	dir := t.TempDir()
	port := startIn(t, dir)
	if err := Stop(); err != nil {
		t.Fatalf("Stop() = %v", err)
	}
	if got := Port(); got != 0 {
		t.Errorf("Port() = %d after stopping", got)
	}
	if conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second); err == nil {
		_ = conn.Close()
		t.Errorf("port %d still answers after stopping", port)
	}
	again, err := Start(dir)
	if err != nil {
		t.Fatalf("starting after stopping: %v", err)
	}
	if again == 0 || Port() != again {
		t.Errorf("started again on %d with Port() = %d", again, Port())
	}
	if err := Stop(); err != nil {
		t.Fatalf("Stop() = %v", err)
	}
	// A server closed by Stop is not one that ended on its own, and says nothing.
	time.Sleep(300 * time.Millisecond)
	if said := read(); strings.Contains(said, "numen:") {
		t.Errorf("stopping was reported as a failure: %q", said)
	}
}

func TestPortIsZeroOnceTheServerEndedOnItsOwn(t *testing.T) {
	startIn(t, t.TempDir())
	mu.Lock()
	running.gone.Store(true)
	mu.Unlock()
	if got := Port(); got != 0 {
		t.Errorf("Port() = %d for a server that ended", got)
	}
}

func TestSeedLeavesAVaultThatHoldsANoteAsItIs(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.md"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := seed(root); err != nil {
		t.Fatal(err)
	}
	held, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(held) != 2 {
		t.Errorf("the vault holds %d entries after seeding, want the 2 it had", len(held))
	}
}

func TestSeedAnswersTheErrorOfAFolderItCannotList(t *testing.T) {
	err := seed(filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("seed answered %v, want the folder's own error", err)
	}
	if strings.Contains(err.Error(), "seeding") {
		t.Errorf("a folder that could not be listed was reported as a failed write: %v", err)
	}
}

func TestRegisterVaultFilesAVaultUnderThePathItResolvesTo(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	cfg := makeConfig(dir, io.Discard)

	filed, err := registerVault(cfg, link)
	if err != nil {
		t.Fatal(err)
	}
	if filed != resolved {
		t.Errorf("filed under %q, want %q", filed, resolved)
	}
	again, err := registerVault(cfg, target)
	if err != nil {
		t.Fatal(err)
	}
	if again != filed {
		t.Errorf("registering the same vault again filed it under %q, want %q", again, filed)
	}
}

func TestRegisterVaultAnswersTheErrorOfAListItCannotRead(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vaults.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	filed, err := registerVault(makeConfig(dir, io.Discard), dir)
	var failed *json.SyntaxError
	if !errors.As(err, &failed) || filed != "" {
		t.Errorf("registerVault answered %q, %v for a list that is not JSON, want no path and the error of reading it", filed, err)
	}
}

func TestRegisterVaultAnswersTheErrorOfAListItCannotWrite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a superuser writes into any folder")
	}
	kept := t.TempDir()
	if err := os.Chmod(kept, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(kept, 0o700) })
	filed, err := registerVault(makeConfig(kept, io.Discard), t.TempDir())
	if !errors.Is(err, fs.ErrPermission) || filed != "" {
		t.Errorf("registerVault answered %q, %v for a list it cannot write, want no path and the permission error", filed, err)
	}
}

func TestAllowOriginAnswersAPreflightWithTheRulesAndNoBody(t *testing.T) {
	reached := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })
	res := httptest.NewRecorder()
	allowOrigin(next).ServeHTTP(res, httptest.NewRequest(http.MethodOptions, "/numen.v1.NoteService/CreateNote", nil))

	if res.Code != http.StatusNoContent {
		t.Errorf("a preflight was answered %d, want %d", res.Code, http.StatusNoContent)
	}
	if reached {
		t.Error("a preflight reached the service")
	}
	checkCORSHeaders(t, res.Header())
}

func TestAllowOriginHandsEveryOtherAskOnWithTheRules(t *testing.T) {
	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusTeapot)
	})
	res := httptest.NewRecorder()
	allowOrigin(next).ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/numen.v1.NoteService/CreateNote", nil))

	if !reached || res.Code != http.StatusTeapot {
		t.Errorf("the ask did not reach the service: reached %v, answered %d", reached, res.Code)
	}
	checkCORSHeaders(t, res.Header())
}

func checkCORSHeaders(t *testing.T, head http.Header) {
	t.Helper()
	if got := head.Get("Access-Control-Allow-Origin"); got != Page {
		t.Errorf("Allow-Origin = %q, want %q", got, Page)
	}
	if got := head.Get("Access-Control-Allow-Methods"); got != "POST, GET, OPTIONS" {
		t.Errorf("Allow-Methods = %q", got)
	}
	if got := head.Get("Access-Control-Max-Age"); got != "86400" {
		t.Errorf("Max-Age = %q", got)
	}
	for name, wanted := range map[string][]string{
		"Access-Control-Allow-Headers": {
			"Content-Type", "Connect-Protocol-Version", "Connect-Timeout-Ms",
			"Connect-Accept-Encoding", "Connect-Content-Encoding", "X-User-Agent",
		},
		"Access-Control-Expose-Headers": {
			"Connect-Content-Encoding", "Connect-Accept-Encoding",
			"Grpc-Status", "Grpc-Message", "Grpc-Status-Details-Bin",
		},
	} {
		listed := map[string]bool{}
		for _, one := range strings.Split(head.Get(name), ", ") {
			listed[one] = true
		}
		for _, one := range wanted {
			if !listed[one] {
				t.Errorf("%s = %q, missing %s", name, head.Get(name), one)
			}
		}
	}
}

func TestSeedAnswersTheErrorOfANoteItCannotWrite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a superuser writes into any folder")
	}
	root := t.TempDir()
	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o700) })
	err := seed(root)
	if !errors.Is(err, fs.ErrPermission) || !strings.Contains(err.Error(), "seeding ") {
		t.Errorf("seed answered %v, want the permission error naming the note it was writing", err)
	}
}

// checkNoIndexOpened holds that a start which ended before it opened the index
// left none behind.
func checkNoIndexOpened(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, "index.db")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a start that failed before the index was opened left it behind: %v", err)
	}
}
