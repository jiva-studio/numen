package process

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func TestProcessPriorityLowersTheChildWithoutChangingTheHost(t *testing.T) {
	before := readNice(t, os.Getpid(), os.Getpid())
	cmd := exec.CommandContext(t.Context(), "sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	target := min(before+3, 19)
	if err := SetNice(cmd.Process.Pid, target); err != nil {
		t.Fatal(err)
	}
	if got := readNice(t, cmd.Process.Pid, cmd.Process.Pid); got != target {
		t.Fatalf("child nice %d, want %d", got, target)
	}
	if got := readNice(t, os.Getpid(), os.Getpid()); got != before {
		t.Fatalf("host nice changed to %d from %d", got, before)
	}
}

func readNice(t *testing.T, pid, tid int) int {
	t.Helper()
	raw, err := os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/stat", pid, tid))
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(string(raw[strings.LastIndexByte(string(raw), ')')+1:]))
	nice, err := strconv.Atoi(fields[16])
	if err != nil {
		t.Fatal(err)
	}
	return nice
}
