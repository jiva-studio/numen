package testsupport

import (
	"context"
	"testing"
	"time"
)

// Patience is how long a test gives something happening in the background
// before it calls the thing broken.
// It is sized for the slowest machine the suite must pass on, a shared Windows
// runner reading a fixture vault while `go test ./...` runs other packages. The
// stream tests of both windows already wait ten seconds, so this is the same number.
const Patience = 10 * time.Second

// WaitFor holds until something is so, and fails the test if it never is.
func WaitFor(t *testing.T, condition func() bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), Patience)
	defer cancel()
	for ctx.Err() == nil {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("it never came to be so")
}
