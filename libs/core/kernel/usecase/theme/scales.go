package theme

import (
	"sync"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
)

// Scales holds the scales given on the command line until the user chooses explicit ones.
type Scales struct {
	mu                 sync.Mutex
	uiScale, textScale float64
}

// NewScales holds the interface and text scales of the command line, zero where none was given.
func NewScales(ui, text float64) *Scales {
	return &Scales{mu: sync.Mutex{}, uiScale: ui, textScale: text}
}

// apply overrides the scales of the appearance with those of the command line, where given.
func (s *Scales) apply(appearance *themedomain.Appearance) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.uiScale > 0 {
		appearance.InterfaceScale = s.uiScale
	}
	if s.textScale > 0 {
		appearance.TextScale = s.textScale
	}
}

// clear drops the command line scales the chosen appearance replaces.
func (s *Scales) clear(chosen themedomain.Appearance) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if chosen.InterfaceScale > 0 {
		s.uiScale = 0
	}
	if chosen.TextScale > 0 {
		s.textScale = 0
	}
}
