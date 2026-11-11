package plugintest

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

// Kill kills the process of cmd and fails the test when the kill fails.
// A process that has already exited is ignored.
func Kill(tb testing.TB, cmd *exec.Cmd) {
	tb.Helper()
	if cmd.Process == nil {
		return
	}
	if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		tb.Errorf("kill: %v", err)
	}
}

// Wait waits for cmd to exit and fails the test when waiting itself fails.
// A non-zero exit status is ignored.
func Wait(tb testing.TB, cmd *exec.Cmd) {
	tb.Helper()
	var exit *exec.ExitError
	if err := cmd.Wait(); err != nil && !errors.As(err, &exit) {
		tb.Errorf("wait: %v", err)
	}
}
