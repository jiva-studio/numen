package recognition

import (
	"testing"
	"time"
)

// Ready is asked by the tool that starts a reading, right after it has started
// one. It answers while that reading is opening the runtime, which on a machine
// holding none is a hundred and thirty megabytes.
func TestReadyAnswersWhileTheRuntimeIsBeingOpened(t *testing.T) {
	held.Lock()
	defer held.Unlock()

	said := make(chan bool, 1)
	go func() { said <- Ready(Defaults()) }()

	select {
	case <-said:
	case <-time.After(2 * time.Second):
		t.Fatal("asking what is on this machine waited for the runtime to be opened")
	}
}
