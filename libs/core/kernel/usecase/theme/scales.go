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

func (s *Scales) clear(choice themedomain.Appearance) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if choice.InterfaceScale > 0 {
		s.uiScale = 0
	}
	if choice.TextScale > 0 {
		s.textScale = 0
	}
}
