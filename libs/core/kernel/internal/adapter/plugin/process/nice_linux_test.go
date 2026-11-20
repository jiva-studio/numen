package process

import (
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func TestProcessPriorityLowersTheChildWithoutChangingTheHost(t *testing.T) {
	before := readNice(t, os.Getpid(), os.Getpid())
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLifetimeHelper$")
	cmd.Env = append(os.Environ(), lifetimeRoleEnv+"="+roleIdlePlugin)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { wiretest.Kill(t, cmd); wiretest.ReapProcess(t, cmd.Process) })
	pid := cmd.Process.Pid
	directory := fmt.Sprintf("/proc/%d/task", pid)
	var threads []os.DirEntry
	wiretest.WaitUntil(t, "plugin runtime threads", func() bool {
		var err error
		threads, err = os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		return len(threads) > 1
	})
	target := min(before+3, 19)
	if err := SetNice(cmd.Process.Pid, target); err != nil {
		t.Fatal(err)
	}
	for _, thread := range threads {
		tid, err := strconv.Atoi(thread.Name())
		if err != nil {
			t.Fatal(err)
		}
		if got := readNice(t, pid, tid); got != target {
			t.Fatalf("thread %d nice %d, want %d", tid, got, target)
		}
	}
	if target > before {
		err := SetNice(pid, before)
		want := before
		switch {
		case errors.Is(err, unix.EACCES), errors.Is(err, unix.EPERM):
			want = target
		case err != nil:
			t.Fatal(err)
		}
		if got := readNice(t, pid, pid); got != want {
			t.Fatalf("raising child priority: nice %d, want %d (error %v)", got, want, err)
		}
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
