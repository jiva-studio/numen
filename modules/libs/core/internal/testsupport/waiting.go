package testsupport

import (
	"testing"
	"time"
)

// Patience is how long a test gives something happening in the background
// before it calls the thing broken.
//
// It is a claim about the slowest machine the suite must pass on, which is a
// shared Windows runner reading a fixture vault while `go test ./...` runs
// other packages beside it. The stream tests of both windows already wait ten
// seconds for a handler to be reached, so this is the number this suite has
// settled on and not a second one.
const Patience = 10 * time.Second

// WaitFor holds until something is so, and fails the test if it never is.
func WaitFor(t *testing.T, so func() bool) {
	t.Helper()
	for at := time.Now(); time.Since(at) < Patience; {
		if so() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("it never came to be so")
}
