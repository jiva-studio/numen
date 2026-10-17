//go:build unix

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
	cmd := exec.CommandContext(running, "sh", "-c", "sleep 120 & echo started; wait")
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

	ended := make(chan error, 1)
	go func() {
		_, err := io.Copy(io.Discard, reader)
		ended <- err
	}()
	select {
	case err := <-ended:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("what the child started outlived it")
	}
	_ = cmd.Wait()
}
