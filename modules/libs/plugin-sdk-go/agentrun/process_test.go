//go:build unix

package agentrun_test

import (
	"bufio"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agentrun"
)

// Cancelling the context of a supervised command ends what the command started as well.
func TestSuperviseEndsWhatTheChildStarted(t *testing.T) {
	running, cancel := context.WithCancel(t.Context())
	defer cancel()
	cmd := exec.CommandContext(running, "sh", "-c", "sleep 120 & echo $!; wait")
	agentrun.Supervise(cmd)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	grandchild, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatal(err)
	}
	if !isRunning(grandchild) {
		t.Fatal("the process the child started is not running")
	}

	cancel()
	_ = cmd.Wait()

	// A process whose parent was killed is waited for by init, so it stays visible for a moment.
	for range 200 {
		if !isRunning(grandchild) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("what the child started outlived it")
}
