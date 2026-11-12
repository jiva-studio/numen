//go:build linux

package performance

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func readNiceValue(t *testing.T, tid string) int {
	t.Helper()
	raw, err := os.ReadFile("/proc/self/task/" + tid + "/stat")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	fields := strings.Fields(text[strings.LastIndexByte(text, ')')+1:])
	nice, err := strconv.Atoi(fields[16])
	if err != nil {
		t.Fatal(err)
	}
	return nice
}

func TestSetNiceLowersThePriorityOfEveryThread(t *testing.T) {
	before := readNiceValue(t, strconv.Itoa(os.Getpid()))
	want := min(before+1, 19)
	if err := SetNice(want); err != nil {
		t.Fatal(err)
	}
	tasks, err := os.ReadDir("/proc/self/task")
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if got := readNiceValue(t, task.Name()); got != want {
			t.Errorf("thread %s has niceness %d, want %d", task.Name(), got, want)
		}
	}
}

func TestSetNiceFallbackAndErrorHandling(t *testing.T) {
	currentNice := readNiceValue(t, strconv.Itoa(os.Getpid()))

	// 1. Non-numeric entry is skipped without calling setPriority(0, -20).
	onlyNonNumeric := t.TempDir()
	if err := os.Mkdir(onlyNonNumeric+"/!not-a-tid", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := setNiceIn(onlyNonNumeric, -20); err != nil {
		t.Fatalf("expected non-numeric task entry to be skipped, got %v", err)
	}

	// 2. Non-numeric entry before valid entry continues the loop, and valid entry succeeds.
	validAfterSkip := t.TempDir()
	for _, name := range []string{"!not-a-tid", strconv.Itoa(os.Getpid())} {
		if err := os.Mkdir(validAfterSkip+"/"+name, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := setNiceIn(validAfterSkip, currentNice); err != nil {
		t.Fatalf("expected setNiceIn(%d) to succeed, got %v", currentNice, err)
	}

	// 3. Non-numeric entry before failing entry continues and returns the first error wrapping ESRCH.
	fakeProc := t.TempDir()
	for _, name := range []string{"!not-a-tid", "999999998", "999999999"} {
		if err := os.Mkdir(fakeProc+"/"+name, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := setNiceIn(fakeProc, currentNice); err == nil || !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("expected ESRCH error for non-existent thread IDs, got %v", err)
	}

	// 4. Fallback when procTaskDir is missing calls setPriority(0, currentNice) and succeeds,
	// and fails when lowering niceness below current without privileges.
	if err := setNiceIn(fakeProc+"/missing", currentNice); err != nil {
		t.Fatalf("expected fallback setPriority(0, %d) to succeed, got %v", currentNice, err)
	}
	if os.Geteuid() != 0 {
		if err := setNiceIn(fakeProc+"/missing", -20); err == nil {
			t.Fatal("expected fallback setNiceIn(-20) to fail for unprivileged user")
		}
	}
}
