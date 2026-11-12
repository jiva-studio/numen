package shared_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/shared"
)

const (
	envHolderDir = "NUMEN_SHARED_TEST_DIR"
	envHolderLog = "NUMEN_SHARED_TEST_LOG"
	holdFor      = 120 * time.Millisecond
)

func hold(dir, log, name string) error {
	release, err := shared.Lock(dir)
	if err != nil {
		return err
	}
	if err := appendLine(log, "enter "+name); err != nil {
		return err
	}
	time.Sleep(holdFor)
	if err := appendLine(log, "leave "+name); err != nil {
		return err
	}
	return release()
}

func appendLine(path, line string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := file.WriteString(line + "\n")
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func assertSerial(t *testing.T, log string, holders int) {
	t.Helper()
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2*holders {
		t.Fatalf("log has %d entries, want %d: %v", len(lines), 2*holders, lines)
	}
	for i := 0; i < len(lines); i += 2 {
		enter, leave := strings.Fields(lines[i]), strings.Fields(lines[i+1])
		if enter[0] != "enter" || leave[0] != "leave" || enter[1] != leave[1] {
			t.Fatalf("holders overlapped: %v", lines)
		}
	}
}

func TestTheHolderOfALockIsAloneInTheDirectory(t *testing.T) {
	dir, log := filepath.Join(t.TempDir(), "digest"), filepath.Join(t.TempDir(), "log")
	var group sync.WaitGroup
	errs := make(chan error, 4)
	for i := range 4 {
		group.Go(func() {
			errs <- hold(dir, log, fmt.Sprint("goroutine-", i))
		})
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	assertSerial(t, log, 4)
}

func TestTwoProcessesCompilingOneDigestTakeTurns(t *testing.T) {
	if os.Getenv(envHolderDir) != "" {
		return
	}
	dir, log := filepath.Join(t.TempDir(), "digest"), filepath.Join(t.TempDir(), "log")
	var commands []*exec.Cmd
	for i := range 3 {
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestHolderProcess$")
		cmd.Env = append(os.Environ(), envHolderDir+"="+dir, envHolderLog+"="+log, "NUMEN_SHARED_TEST_NAME="+fmt.Sprint("process-", i))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, cmd)
	}
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			t.Fatal(err)
		}
	}

	assertSerial(t, log, 3)
}

// TestHolderProcess is the body of the processes TestTwoProcessesCompilingOneDigestTakeTurns starts.
func TestHolderProcess(t *testing.T) {
	dir := os.Getenv(envHolderDir)
	if dir == "" {
		t.Skip("runs only as a child of the test that starts it")
	}
	if err := hold(dir, os.Getenv(envHolderLog), os.Getenv("NUMEN_SHARED_TEST_NAME")); err != nil {
		t.Fatal(err)
	}
}

func TestALockedDirectoryIsCreatedForTheUserAlone(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "digest")

	release, err := shared.Lock(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := release(); err != nil {
			t.Error(err)
		}
	}()

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("stat %s = %v, %v", dir, info, err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 {
		t.Errorf("mode = %o, want 700", info.Mode().Perm())
	}
	lockInfo, err := os.Stat(filepath.Join(dir, ".lock"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && lockInfo.Mode().Perm() != 0o600 {
		t.Errorf("lock file mode = %o, want 600", lockInfo.Mode().Perm())
	}

	blockingFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := shared.Lock(filepath.Join(blockingFile, "sub")); err == nil || strings.HasPrefix(err.Error(), "lock ") {
		t.Errorf("expected direct MkdirAll error without lock prefix, got %v", err)
	}

	badLockDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(badLockDir, ".lock"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := shared.Lock(badLockDir); err == nil || !strings.HasPrefix(err.Error(), "lock ") || !errors.Is(err, syscall.EISDIR) {
		t.Errorf("expected wrapped EISDIR lock error when .lock is a directory, got %v", err)
	}
}

func TestAModelTheHostProvisionedIsKeyedByTheSumItWasVerifiedAgainst(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "encoder.onnx")
	sum := strings.Repeat("ab", 32)
	for path, content := range map[string]string{model: "weights", filepath.Join(dir, ".ready"): "encoder.onnx\n" + sum + "\n"} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	key, err := shared.GetModelKey(model)

	if err != nil || key != sum {
		t.Errorf("GetModelKey() = %q, %v, want %q", key, err, sum)
	}
}

func TestAModelTheHostDidNotProvisionIsKeyedByItsStat(t *testing.T) {
	model := filepath.Join(t.TempDir(), "own.onnx")
	if err := os.WriteFile(model, []byte("weights"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := shared.GetModelKey(model)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(model, []byte("other weights"), 0o600); err != nil {
		t.Fatal(err)
	}

	second, err := shared.GetModelKey(model)

	if err != nil || first == second || !strings.HasPrefix(first, "stat-") {
		t.Errorf("keys %q and %q, %v, want two different stat keys", first, second, err)
	}

	if _, err := shared.GetModelKey(filepath.Join(t.TempDir(), "missing.onnx")); err == nil {
		t.Fatal("expected error for missing model file")
	}

	sum := strings.Repeat("cd", 32)
	readyPath := filepath.Join(filepath.Dir(model), ".ready")
	for _, invalidReady := range []string{
		"own.onnx\n",
		"other.onnx\n" + sum + "\n",
		"own.onnx\nabcd\n",
		"own.onnx\n" + strings.Repeat("zz", 32) + "\n",
	} {
		if err := os.WriteFile(readyPath, []byte(invalidReady), 0o600); err != nil {
			t.Fatal(err)
		}
		key, err := shared.GetModelKey(model)
		if err != nil || !strings.HasPrefix(key, "stat-") {
			t.Errorf("ready %q: got key=%q err=%v, want stat- key", invalidReady, key, err)
		}
	}
}
