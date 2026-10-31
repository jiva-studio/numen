//go:build unix

package supervise

import (
	"context"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestCancellingTheContextEndsTheWholeProcessGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cmd := exec.CommandContext(ctx, "sh", "-c", "sleep 30 & echo $!; wait")
	Command(cmd)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var childPID int
	buf := make([]byte, 32)
	n, err := out.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range buf[:n] {
		if c >= '0' && c <= '9' {
			childPID = childPID*10 + int(c-'0')
		}
	}

	cancel()
	if err := cmd.Wait(); err == nil {
		t.Error("Wait() = nil, want the error of a killed command")
	}

	for range 500 {
		if syscall.Kill(childPID, 0) != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("process %d of the group is still alive", childPID)
}
