//go:build windows

package agentrun_test

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agentrun"
)

// Cancelling the context of a supervised command ends what the command started as well. The output
// of the child is held by everything it started, so it ends only when all of them have ended.
func TestSuperviseEndsWhatTheChildStarted(t *testing.T) {
	running, cancel := context.WithCancel(t.Context())
	defer cancel()
	cmd := exec.CommandContext(running, "cmd", "/c", "echo started & ping -n 120 127.0.0.1")
	agentrun.Supervise(cmd)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(out)
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatal(err)
	}
	cancel()

	ended := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, reader)
		close(ended)
	}()
	select {
	case <-ended:
	case <-time.After(30 * time.Second):
		t.Fatal("what the child started outlived it")
	}
	_ = cmd.Wait()
}
