package agents

import (
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// serving says whether the tools are in front of the agents.
func (s *Endpoint) serving() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shut != nil
}

// standing is a window on a vault, serving tools that do nothing.
func standing() *Endpoint {
	return &Endpoint{
		Serve:       func() (func() error, error) { return func() error { return nil }, nil },
		Standing:    func() domain.Vault { return domain.Vault{ID: "one"} },
		Answers:     func(port.Agent) {},
		Unreachable: func(string) {},
		Trouble:     func(error) {},
	}
}

// TestOneSwapHoldsTheAgentsUntilItIsOver. The tools are served for the vault in
// the window, so a second swap arriving while one runs does not put them back
// in front of the agents on the vault that is going.
func TestOneSwapHoldsTheAgentsUntilItIsOver(t *testing.T) {
	s := standing()
	s.On()
	if !s.serving() {
		t.Fatal("the tools were never served")
	}

	running := make(chan struct{})
	release := make(chan struct{})

	var swaps sync.WaitGroup
	swaps.Add(1)
	go func() {
		defer swaps.Done()
		_ = s.Around(func() error {
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
		_ = s.Around(func() error { return nil })
	}()
	// The second swap has this long to reach the endpoint the first is holding.
	select {
	case <-second:
	case <-time.After(time.Second):
	}
	served := s.serving()

	close(release)
	swaps.Wait()

	if served {
		t.Error("the tools were served again while a swap was running")
	}
	if !s.serving() {
		t.Error("the tools were not served again once the swap was over")
	}
}
