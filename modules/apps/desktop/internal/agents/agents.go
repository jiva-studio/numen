// Package agents is how a window lets an agent reach the vault it is showing.
//
// The tools go on a port, and the token an agent presents is kept between launches.
// A window says here what an agent may reach.
package agents

import (
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// What an agent is told about the vault it is working is said once, when its
// session opens, so a window that changes vault stops the endpoint and starts
// it again.
type Endpoint struct {
	// Serve puts the tools in front of the agents, and answers with what takes
	// them away again.
	Serve func() (func() error, error)
	// GetVault is the vault the window is on. A window standing on none serves
	// no tools.
	GetVault func() domain.Vault
	// SetUnreachable is told why an agent cannot be reached, and an empty string
	// while one can be.
	SetUnreachable func(string)
	// ErrorHandler is told what went wrong serving the tools or taking them away.
	ErrorHandler func(error)

	// runMu is one swap. It is held from the endpoint stopping to the endpoint
	// being served again, so a second swap waits for the first.
	runMu sync.Mutex

	mu    sync.Mutex
	close func() error
}

// IsServing reports whether the tools are currently served.
func (s *Endpoint) IsServing() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.close != nil
}

// RunWhileStopped runs one swap with the tools taken away, and serves them again on the
// vault the window then has.
//
// One swap holds this at a time, so the endpoint is started again by the swap
// that stopped it and on the vault that swap ended on.
func (s *Endpoint) RunWhileStopped(swap func() error) error {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	s.Stop()
	defer s.Start()
	return swap()
}

// Start serves the tools against the vault in the window. A window standing on
// no vault serves none.
func (s *Endpoint) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.close != nil || s.GetVault().ID == "" {
		return
	}
	s.SetUnreachable("")
	shut, err := s.Serve()
	if err != nil {
		s.SetUnreachable(err.Error())
		s.ErrorHandler(fmt.Errorf("no agent: %w", err))
		return
	}
	s.close = shut
}

// Stop stops the endpoint this window started.
func (s *Endpoint) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.close == nil {
		return
	}
	if err := s.close(); err != nil {
		s.ErrorHandler(fmt.Errorf("agents: %w", err))
	}
	s.close = nil
}
