package agents

import (
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// newEndpoint is a window on a vault, serving tools that do nothing.
func newEndpoint() *Endpoint {
	return &Endpoint{
		Serve:          func() (func() error, error) { return func() error { return nil }, nil },
		GetVault:       func() domain.Vault { return domain.Vault{ID: "one"} },
		SetUnreachable: func(string) {},
		ErrorHandler:   func(error) {},
	}
}

// TestRunWhileStopped_AgentsRemainStopped. The tools are served for the vault in
// the window, so a second swap arriving while one runs does not put them back
// in front of the agents on the vault that is going.
func TestRunWhileStopped_AgentsRemainStopped(t *testing.T) {
	s := newEndpoint()
	s.Start()
	if !s.IsServing() {
		t.Fatal("the tools were never served")
	}

	running := make(chan struct{})
	release := make(chan struct{})

	var swaps sync.WaitGroup
	swaps.Add(1)
	go func() {
		defer swaps.Done()
		_ = s.RunWhileStopped(func() error {
			close(running)
			<-release
			return nil
		})
	}()
	<-running

	second := make(chan struct{})
	swaps.Add(1)
	go func() {
		defer swaps.Done()
		defer close(second)
		_ = s.RunWhileStopped(func() error { return nil })
	}()
	// The second swap has this long to reach the endpoint the first is holding.
	select {
	case <-second:
	case <-time.After(time.Second):
	}
	served := s.IsServing()

	close(release)
	swaps.Wait()

	if served {
		t.Error("the tools were served again while a swap was running")
	}
	if !s.IsServing() {
		t.Error("the tools were not served again once the swap was over")
	}
}
