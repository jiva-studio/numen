// Package errsink holds the handler that receives the problems an adapter meets outside a call, such
// as a failed shutdown.
package errsink

import "sync"

// Sink hands a problem to the handler set last and drops it where none is set.
type Sink struct {
	mu      sync.RWMutex
	handler func(error)
}

// Set sets the handler that receives the problems.
func (s *Sink) Set(handler func(error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = handler
}

// Report hands the problem to the handler.
func (s *Sink) Report(err error) {
	s.mu.RLock()
	handler := s.handler
	s.mu.RUnlock()
	if handler != nil {
		handler(err)
	}
}
