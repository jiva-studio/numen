//go:build linux

package performance

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

// readNice is the niceness of the thread tid, field 19 of its stat file.
func readNice(t *testing.T, tid string) int {
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
	before := readNice(t, strconv.Itoa(os.Getpid()))
	want := min(before+1, 19)
	if err := SetNice(want); err != nil {
		t.Fatal(err)
	}
	tasks, err := os.ReadDir("/proc/self/task")
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if got := readNice(t, task.Name()); got != want {
			t.Errorf("thread %s has niceness %d, want %d", task.Name(), got, want)
		}
	}
}
