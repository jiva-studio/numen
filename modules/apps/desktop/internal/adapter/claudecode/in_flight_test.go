package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// counting is a program standing in for the command line that records how many
// of itself run at once. It holds its turn long enough for the others to start.
func counting(t *testing.T) (command []string, most func() int) {
	t.Helper()
	dir := t.TempDir()
	at := filepath.Join(dir, "now")
	peak := filepath.Join(dir, "most")
	script := filepath.Join(dir, "counting")
	written := "#!/bin/sh\n" +
		"exec 9>" + dir + "/lock\n" +
		"flock 9; n=$(cat " + at + " 2>/dev/null || echo 0); n=$((n+1)); echo $n > " + at + "\n" +
		"m=$(cat " + peak + " 2>/dev/null || echo 0); [ $n -gt $m ] && echo $n > " + peak + "\n" +
		"flock -u 9\n" +
		"cat > /dev/null; sleep 0.3\n" +
		"flock 9; n=$(cat " + at + "); echo $((n-1)) > " + at + "; flock -u 9\n" +
		"printf ''\n"
	if err := os.WriteFile(script, []byte(written), 0o755); err != nil {
		t.Fatal(err)
	}
	return []string{script}, func() int {
		raw, err := os.ReadFile(peak)
		if err != nil {
			t.Fatal(err)
		}
		n, err := strconv.Atoi(string(raw[:len(raw)-1]))
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
}

func batches(n int) []proofread.Batch {
	out := make([]proofread.Batch, n)
	for i := range out {
		out[i] = proofread.Batch{Number: i, Lines: []proofread.Line{{Number: i, Text: "a line"}}}
	}
	return out
}

// A person's own model is left most of itself: no more runs stand at once than
// the profile allows.
func TestNoMoreRunsStandAtOnceThanWereAllowed(t *testing.T) {
	for _, allowed := range []int{1, 2, 3} {
		command, most := counting(t)
		by := &Proofreader{Command: command, Instruction: "put it right", InFlight: allowed}

		if _, err := by.Proofread(context.Background(), batches(6)); err != nil {
			t.Fatal(err)
		}
		if got := most(); got > allowed {
			t.Errorf("%d allowed, %d stood at once", allowed, got)
		} else if got < allowed {
			t.Errorf("%d allowed, only %d ever stood at once", allowed, got)
		}
	}
}

// A profile naming none runs one at a time.
func TestAProfileNamingNoneRunsOneAtATime(t *testing.T) {
	command, most := counting(t)
	by := &Proofreader{Command: command, Instruction: "put it right"}

	if _, err := by.Proofread(context.Background(), batches(4)); err != nil {
		t.Fatal(err)
	}
	if got := most(); got != 1 {
		t.Errorf("%d stood at once", got)
	}
}

// The limit is the proofreader's and not one run's: two callers asking at once
// take no more of a person's own model between them than the profile allows.
func TestTwoCallersAtOnceShareTheLimit(t *testing.T) {
	command, most := counting(t)
	by := &Proofreader{Command: command, Instruction: "put it right", InFlight: 2}

	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := by.Proofread(context.Background(), batches(3)); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	if got := most(); got > 2 {
		t.Errorf("2 allowed, %d stood at once", got)
	}
}
