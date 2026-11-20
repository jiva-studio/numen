package process

import (
	"errors"
	"os"
	"testing"
)

func TestPriorityChangesAreAttemptedOncePerProcessAndValue(t *testing.T) {
	var scheduler Child
	first := &os.Process{Pid: 1}
	second := &os.Process{Pid: 2}
	refused := errors.New("raise denied")
	calls := 0
	apply := func(_ int, nice int) error {
		calls++
		if nice == 0 {
			return refused
		}
		return nil
	}
	for _, test := range []struct {
		child *os.Process
		nice  int
		calls int
		err   error
	}{
		{nil, 10, 0, nil}, {first, 10, 1, nil}, {first, 19, 2, nil}, {first, 0, 3, refused}, {first, 0, 3, nil}, {second, 0, 4, refused}, {second, 10, 5, nil},
	} {
		scheduler.SetProcess(test.child)
		if err := scheduler.applyNice(test.nice, apply); !errors.Is(err, test.err) || calls != test.calls {
			t.Fatalf("nice %d: calls %d, error %v", test.nice, calls, err)
		}
	}
}
