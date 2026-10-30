package testsupport

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
)

// Close closes c and fails the test when closing returns an error.
func Close(tb testing.TB, c io.Closer) {
	tb.Helper()
	if err := c.Close(); err != nil {
		tb.Errorf("close: %v", err)
	}
}

// CloseAfter closes c when the test ends and fails the test when closing returns an error.
func CloseAfter(tb testing.TB, c io.Closer) {
	tb.Helper()
	tb.Cleanup(func() { Close(tb, c) })
}

// Kill kills the process of cmd and fails the test when the kill fails.
// A process that has already exited is not a failure.
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
// A non-zero exit status is not a failure: it is what a killed process reports.
func Wait(tb testing.TB, cmd *exec.Cmd) {
	tb.Helper()
	var exit *exec.ExitError
	if err := cmd.Wait(); err != nil && !errors.As(err, &exit) {
		tb.Errorf("wait: %v", err)
	}
}
