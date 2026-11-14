package process

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// initializeFixture is the answer of Claude Code 2.1.303 to the initialize request, cut to its models.
const initializeFixture = "../../../model/testdata/initialize-response.json"

func readAnswer(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(initializeFixture)
	if err != nil {
		t.Fatal(err)
	}
	answer := strings.Replace(string(raw), `"r1"`, `"numen-models"`, 1)
	return strings.ReplaceAll(answer, "'", "")
}

func writeScript(t *testing.T, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fixtures are shell scripts")
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

// The program receives the request over stdin and its answer carries the request id.
func TestListModelsAsksTheProgramOverItsInput(t *testing.T) {
	request := t.TempDir() + "/asked"
	script := writeScript(t, "claude", "read line\nprintf '%s\\n' \"$line\" > "+request+"\ncat <<'EOF'\n"+readAnswer(t)+"EOF\nsleep 30\n")

	list, err := (Lister{}).ListModels(t.Context(), []string{script}, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 13 {
		t.Fatalf("the models are %v", list)
	}
	raw, err := os.ReadFile(request)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"subtype":"initialize"`) {
		t.Errorf("the program was asked %s", raw)
	}
}

// A program that exits as soon as it has answered lists its models and not the fallback: on macOS the
// kill of its exited process group fails.
func TestAProgramThatExitsAfterAnsweringListsItsModels(t *testing.T) {
	script := writeScript(t, "claude", "read -r _\ncat <<'EOF'\n"+readAnswer(t)+"EOF\nexit 0\n")

	list, err := (Lister{}).ListModels(t.Context(), []string{script}, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 13 {
		t.Errorf("the models are %v", list)
	}
}

func TestIsStopped(t *testing.T) {
	if !isStopped(nil) {
		t.Error("expected nil error to be stopped")
	}
	if !isStopped(&exec.ExitError{}) {
		t.Error("expected ExitError to be stopped")
	}
	if !isStopped(context.Canceled) {
		t.Error("expected context.Canceled to be stopped")
	}
	if !isStopped(os.ErrProcessDone) {
		t.Error("expected os.ErrProcessDone to be stopped")
	}
	if !isStopped(syscall.ESRCH) {
		t.Error("expected syscall.ESRCH to be stopped")
	}
	if isStopped(errors.New("other error")) {
		t.Error("expected arbitrary error to not be stopped")
	}
}

func TestListModelsReportsAProgramThatCannotStart(t *testing.T) {
	if _, err := (Lister{}).ListModels(t.Context(), []string{"/nonexistent-claude-bin"}, nil); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ListModels nonexistent = %v, want os.ErrNotExist", err)
	}
}

// Command arguments after command[0] and the environment without the session variables reach the
// program.
func TestListModelsPassesTheArgumentsAndTheSanitizedEnvironment(t *testing.T) {
	body := "if [ \"$1\" != \"--extra\" ] || [ -n \"$CLAUDECODE\" ] || [ \"$KEEP_VAR\" != \"yes\" ]; then exit 2; fi\n" +
		"read -r _\ncat <<'EOF'\n" + readAnswer(t) + "EOF\nexit 0\n"
	list, err := (Lister{}).ListModels(t.Context(), []string{"/bin/sh", "-c", body, "sh", "--extra"},
		[]string{"PATH=" + os.Getenv("PATH"), "CLAUDECODE=1", "KEEP_VAR=yes"})
	if err != nil || len(list) != 13 {
		t.Fatalf("ListModels with args and env = (%d, %v)", len(list), err)
	}
}

func TestSuperviseAndWaitForExitKillProcessGroup(t *testing.T) {
	empty := &exec.Cmd{}
	Supervise(empty)
	if empty.WaitDelay != 2*time.Second || ExitGrace != time.Second {
		t.Fatalf("WaitDelay = %v, ExitGrace = %v", empty.WaitDelay, ExitGrace)
	}
	if empty.SysProcAttr == nil || !empty.SysProcAttr.Setpgid {
		t.Fatalf("SysProcAttr = %+v, want Setpgid=true", empty.SysProcAttr)
	}
	if err := empty.Cancel(); err != nil {
		t.Errorf("Cancel on unstarted command = %v, want nil", err)
	}

	pidFile := filepath.Join(t.TempDir(), "child.pid")
	script := writeScript(t, "hang", "sleep 30 &\necho $! > "+pidFile+"\nwait\n")
	cmd := exec.CommandContext(t.Context(), script)
	Supervise(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	var rawPID []byte
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(pidFile); err == nil && strings.TrimSpace(string(data)) != "" {
			rawPID = data
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	childPID, err := strconv.Atoi(strings.TrimSpace(string(rawPID)))
	if err != nil {
		t.Fatalf("read child pid %q: %v", rawPID, err)
	}

	var stopErr error
	waitErr := WaitForExit(cmd, func() { stopErr = cmd.Cancel() })
	if waitErr == nil {
		t.Error("expected killed command to return non-nil Wait error")
	}
	if stopErr != nil {
		t.Errorf("first Cancel = %v, want nil", stopErr)
	}
	if err := syscall.Kill(childPID, 0); !errors.Is(err, syscall.ESRCH) {
		t.Errorf("expected background child in process group to be killed, Kill(0) = %v", err)
	}
	if err := cmd.Cancel(); !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("Cancel on exited process = %v, want os.ErrProcessDone", err)
	}
}
