// Package agents is how a window lets an agent reach the vault it is showing.
//
// The tools go on a port, the token an agent presents is kept between launches,
// and the agent the settings name is started against them. A window says here
// what an agent may reach.
package agents

import (
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Swapping is the agents' endpoint on the vault a window is showing.
//
// What an agent is told about the vault it is working is said once, when its
// session opens, so a window that changes vault stops the endpoint and starts
// it again.
type Swapping struct {
	// Serve puts the tools in front of the agents, and answers with what takes
	// them away again.
	Serve func() (func() error, error)
	// Standing is the vault the window is on. A window standing on none serves
	// no tools.
	Standing func() domain.Vault
	// Answers is who the panel's tasks go to, and nothing while the tools are
	// away.
	Answers func(port.Agent)
	// Unreachable is told why an agent cannot be reached, and an empty string
	// while one can be.
	Unreachable func(string)
	// Trouble is told what went wrong serving the tools or taking them away.
	Trouble func(error)

	// turn is one swap. It is held from the endpoint stopping to the endpoint
	// being served again, so a second swap waits for the first.
	turn sync.Mutex

	mu   sync.Mutex
	shut func() error
}

// Around runs one swap with the tools taken away, and serves them again on the
// vault the window then has.
//
// One swap holds this at a time, so the endpoint is started again by the swap
// that stopped it and on the vault that swap ended on.
func (s *Swapping) Around(swap func() error) error {
	s.turn.Lock()
	defer s.turn.Unlock()

	s.Off()
	defer s.On()
	return swap()
}

// On serves the tools against the vault in the window. A window standing on no
// vault serves none.
func (s *Swapping) On() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shut != nil || s.Standing().ID == "" {
		return
	}
	s.Unreachable("")
	shut, err := s.Serve()
	if err != nil {
		s.Unreachable(err.Error())
		s.Trouble(fmt.Errorf("no agent: %w", err))
		return
	}
	s.shut = shut
}

// Off stops the endpoint and the agents this window started.
func (s *Swapping) Off() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shut == nil {
		return
	}
	if err := s.shut(); err != nil {
		s.Trouble(fmt.Errorf("agents: %w", err))
	}
	s.shut = nil
	s.Answers(nil)
}
