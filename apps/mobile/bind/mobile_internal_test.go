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
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/jiva-studio/numen/libs/core/container"
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
	isDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&collected, reader)
		close(isDone)
	}()
	return func() string {
		os.Stderr = previous
		_ = writer.Close()
		<-isDone
		_ = reader.Close()
		return collected.String()
	}
}

func setUmask(t *testing.T, mask int) {
	t.Helper()
	previous := syscall.Umask(mask)
	t.Cleanup(func() { syscall.Umask(previous) })
}

func TestStart_WhenAlreadyRunning_ReturnsAssignedPort(t *testing.T) {
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
	var sysErr *os.SyscallError
	if !errors.As(err, &sysErr) || sysErr.Syscall != "setenv" || port != 0 {
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

func TestStart_ThemesFolderCreationFailure_LogsAndStarts(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "themes"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	read := captureStderr(t)
	startIn(t, dir)
	if response := read(); !strings.Contains(response, "themes:") {
		t.Errorf("nothing said the themes folder could not be made: %q", response)
	}
}

func TestStart_ThemesFolderExists_NoThemesErrorLogged(t *testing.T) {
	read := captureStderr(t)
	startIn(t, t.TempDir())
	if response := read(); strings.Contains(response, "themes:") {
		t.Errorf("themes were complained of: %q", response)
	}
}

func TestStart_InstallsThemesForWindow(t *testing.T) {
	startIn(t, t.TempDir())
	mu.Lock()
	defer mu.Unlock()
	if running.opened.Services.Themes == nil {
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
	if response := read(); strings.Contains(response, "numen:") {
		t.Errorf("stopping was reported as a failure: %q", response)
	}
}

func TestPortIsZeroOnceTheServerEndedOnItsOwn(t *testing.T) {
	startIn(t, t.TempDir())
	mu.Lock()
	running.isDown.Store(true)
	mu.Unlock()
	if got := Port(); got != 0 {
		t.Errorf("Port() = %d for a server that ended", got)
	}
}

func TestSeed_PreservesExistingVaultNotes(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.md"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := seedVault(root); err != nil {
		t.Fatal(err)
	}
	target, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(target) != 2 {
		t.Errorf("the vault holds %d entries after seeding, want the 2 it had", len(target))
	}
}

func TestSeedAnswersTheErrorOfAFolderItCannotList(t *testing.T) {
	err := seedVault(filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("seed answered %v, want the folder's own error", err)
	}
	if strings.Contains(err.Error(), "seeding") {
		t.Errorf("a folder that could not be listed was reported as a failed write: %v", err)
	}
}

func TestRegisterVault_RegistersVaultAtResolvedPath(t *testing.T) {
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

func TestRegisterVaultFindsVaultWhenGivenSymlinkToExistingVault(t *testing.T) {
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

	first, err := registerVault(cfg, target)
	if err != nil {
		t.Fatal(err)
	}
	if first != resolved {
		t.Errorf("first registration filed under %q, want %q", first, resolved)
	}

	second, err := registerVault(cfg, link)
	if err != nil {
		t.Fatalf("registering link to existing vault failed: %v", err)
	}
	if second != resolved {
		t.Errorf("second registration filed under %q, want %q", second, resolved)
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
	tempDir := t.TempDir()
	if err := os.Chmod(tempDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(tempDir, 0o700) })
	filed, err := registerVault(makeConfig(tempDir, io.Discard), t.TempDir())
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

func TestAllowOrigin_ForwardsRequestsWithHeaders(t *testing.T) {
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
		items := map[string]bool{}
		for _, one := range strings.Split(head.Get(name), ", ") {
			items[one] = true
		}
		for _, one := range wanted {
			if !items[one] {
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
	err := seedVault(root)
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

func TestStart_SeedError_FailsAndCleansUp(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a superuser writes into any folder")
	}
	keepEnvironment(t)
	dir := t.TempDir()
	root := filepath.Join(dir, "vault")
	if err := os.Mkdir(root, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o700) })
	port, err := Start(dir)
	if err == nil || !errors.Is(err, fs.ErrPermission) || !strings.Contains(err.Error(), "seeding ") || port != 0 {
		_ = Stop()
		t.Fatalf("Start answered %d, %v for a vault folder it cannot write into, want 0 and the error of seeding", port, err)
	}
	checkNoIndexOpened(t, dir)
	if Port() != 0 {
		t.Errorf("a start that failed listens on %d", Port())
	}
}

// feedVaultList stands a pipe in place of the vault list. Every read of it is
// answered with the list as it was while the index is not yet there, and with
// text that is no JSON once it is.
func feedVaultList(t *testing.T, dir string, list []byte) {
	t.Helper()
	path := filepath.Join(dir, "vaults.json")
	index := filepath.Join(dir, "index.db")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	isDone := make(chan struct{})
	go func() {
		for {
			select {
			case <-isDone:
				return
			default:
			}
			pipe, err := os.OpenFile(path, os.O_WRONLY, 0)
			if err != nil {
				return
			}
			body := list
			if _, err := os.Stat(index); err == nil {
				body = []byte("{not json")
			}
			_, _ = pipe.Write(body)
			_ = pipe.Close()
		}
	}()
	t.Cleanup(func() {
		close(isDone)
		if release, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = release.Close()
		}
	})
}

func TestStartAnswersTheErrorOfAVaultListThatCannotBeReadOnceTheIndexIsOpen(t *testing.T) {
	dir := t.TempDir()
	startIn(t, dir)
	if err := Stop(); err != nil {
		t.Fatalf("Stop() = %v", err)
	}
	list, err := os.ReadFile(filepath.Join(dir, "vaults.json"))
	if err != nil {
		t.Fatal(err)
	}
	indexes, err := filepath.Glob(filepath.Join(dir, "index.db*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range indexes {
		if err := os.Remove(name); err != nil {
			t.Fatal(err)
		}
	}
	feedVaultList(t, dir, list)

	port, err := Start(dir)
	var failed *json.SyntaxError
	if !errors.As(err, &failed) || port != 0 {
		_ = Stop()
		t.Fatalf("Start answered %d, %v for a vault list that stopped being JSON, want 0 and the error of reading it", port, err)
	}
	if Port() != 0 {
		t.Errorf("a start that failed listens on %d", Port())
	}
}

// countOpenFiles is the lowest unused file descriptor in the process.
func countOpenFiles() int {
	for fd := 0; fd < 4096; fd++ {
		if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0); errno != 0 {
			return fd
		}
	}
	return 4096
}

func TestStart_SocketError_FailsAndCleansUp(t *testing.T) {
	keepEnvironment(t)
	var before syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &before); err != nil {
		t.Skipf("the open files limit cannot be read: %v", err)
	}
	restore := func() { _ = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &before) }
	t.Cleanup(restore)
	for extra := 0; extra < 200; extra++ {
		dir := t.TempDir()
		limit := before
		limit.Cur = uint64(countOpenFiles() + extra)
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
			t.Skipf("the open files limit cannot be lowered: %v", err)
		}
		port, err := Start(dir)
		restore()
		if err == nil {
			_ = Stop()
			t.Fatalf("Start answered %d with %d files to spare, and was never denied a socket", port, extra)
		}
		var failed *net.OpError
		if !errors.As(err, &failed) || failed.Op != "listen" {
			continue
		}
		if !errors.Is(err, syscall.EMFILE) || port != 0 {
			t.Fatalf("Start answered %d, %v, want 0 and the error of listening", port, err)
		}
		if Port() != 0 {
			t.Errorf("a start that failed listens on %d", Port())
		}
		return
	}
	t.Fatal("no limit on open files left Start failed at the socket alone")
}

func TestStop_CancelsCoreContext(t *testing.T) {
	startIn(t, t.TempDir())
	mu.Lock()
	cancel := running.stop
	cancelled := false
	running.stop = func() {
		cancelled = true
		cancel()
	}
	mu.Unlock()
	if err := Stop(); err != nil {
		t.Fatalf("Stop() = %v", err)
	}
	if !cancelled {
		t.Error("stopping left what the core runs under standing")
	}
}

// refusingListener is a listener that reports its close as failed, once the
// server using it is accepting on it.
type refusingListener struct {
	net.Listener
	err      error
	accepted chan struct{}
	once     *sync.Once
}

func (l refusingListener) Accept() (net.Conn, error) {
	l.once.Do(func() { close(l.accepted) })
	return l.Listener.Accept()
}

func (l refusingListener) Close() error {
	_ = l.Listener.Close()
	return l.err
}

func TestStopAnswersTheErrorOfAServerThatWillNotClose(t *testing.T) {
	startIn(t, t.TempDir())
	inner, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closeErr := errors.New("the listener will not close")
	accepted := make(chan struct{})
	stubborn := &http.Server{}
	go func() { _ = stubborn.Serve(refusingListener{inner, closeErr, accepted, &sync.Once{}}) }()
	<-accepted

	mu.Lock()
	live := running.server
	running.server = stubborn
	mu.Unlock()
	t.Cleanup(func() { _ = live.Close() })

	if err := Stop(); !errors.Is(err, closeErr) {
		t.Errorf("Stop() = %v, want the error of closing the server", err)
	}
	if Port() != 0 {
		t.Errorf("a core that was stopped listens on %d", Port())
	}
}

func TestRegisterVaultFilesAVaultReachedByALinkOnceWhicheverWayItIsAskedFor(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	cfg := makeConfig(dir, io.Discard)
	first, err := registerVault(cfg, link)
	if err != nil {
		t.Fatal(err)
	}
	second, err := registerVault(cfg, link)
	if err != nil {
		t.Fatalf("registering the same link again: %v", err)
	}
	if second != first {
		t.Errorf("registering the same link again filed it under %q, want %q", second, first)
	}
}

func TestRegisterVaultAnswersTheErrorOfAListItCannotReadBeforeItLooksAtTheFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vaults.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	filed, err := registerVault(makeConfig(dir, io.Discard), filepath.Join(dir, "missing"))
	var failed *json.SyntaxError
	if !errors.As(err, &failed) || filed != "" {
		t.Errorf("registerVault answered %q, %v for a list that is not JSON, want no path and the error of reading it", filed, err)
	}
}

func TestRegisterVaultAnswersTheErrorOfAMachineWithNoPlaceForTheList(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	filed, err := registerVault(container.Config{}, t.TempDir())
	if err == nil || filed != "" {
		t.Errorf("registerVault answered %q, %v for a machine with no configuration folder, want no path and an error", filed, err)
	}
}

// getListenersOf is the listeners a server is accepting on. The server holds
// them in an unexported set, under an unexported lock.
func getListenersOf(server *http.Server) []net.Listener {
	elem := reflect.ValueOf(server).Elem()
	lockField := elem.FieldByName("mu")
	lock := reflect.NewAt(lockField.Type(), unsafe.Pointer(lockField.UnsafeAddr())).Interface().(*sync.Mutex)
	setField := elem.FieldByName("listeners")
	target := reflect.NewAt(setField.Type(), unsafe.Pointer(setField.UnsafeAddr())).Elem()
	lock.Lock()
	defer lock.Unlock()
	var found []net.Listener
	for _, key := range target.MapKeys() {
		found = append(found, *(*net.Listener)(key.UnsafePointer()))
	}
	return found
}

// closeListenersOf closes the listeners a server is accepting on, without the
// server being told to stop.
func closeListenersOf(t *testing.T, server *http.Server) {
	t.Helper()
	var found []net.Listener
	for until := time.Now().Add(5 * time.Second); len(found) == 0 && time.Now().Before(until); {
		found = getListenersOf(server)
		time.Sleep(10 * time.Millisecond)
	}
	if len(found) == 0 {
		t.Fatal("the server is accepting on no listener")
	}
	for _, one := range found {
		if err := one.Close(); err != nil {
			t.Fatalf("closing the listener: %v", err)
		}
	}
}

func TestPortIsZeroAndTheErrorIsSaidOnceTheServerStopsAcceptingOnItsOwn(t *testing.T) {
	read := captureStderr(t)
	startIn(t, t.TempDir())
	mu.Lock()
	server := running.server
	mu.Unlock()
	closeListenersOf(t, server)

	until := time.Now().Add(5 * time.Second)
	for Port() != 0 && time.Now().Before(until) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := Port(); got != 0 {
		t.Errorf("Port() = %d for a server that stopped accepting", got)
	}
	if err := Stop(); err != nil {
		t.Logf("Stop() = %v", err)
	}
	if response := read(); !strings.Contains(response, "numen: accept") {
		t.Errorf("nothing said the server stopped accepting: %q", response)
	}
}
