package theme

import "sync"

// Scales holds CLI scale overrides until explicitly changed.
type Scales struct {
	mu                 sync.Mutex
	uiScale, textScale float64
}

// NewScales creates CLI scale overrides.
func NewScales(ui, text float64) *Scales {
	return &Scales{uiScale: ui, textScale: text}
}

// Apply overrides appearance scales with CLI flags if set.
func (l *Scales) Apply(appearance *Appearance) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.uiScale > 0 {
		appearance.InterfaceScale = l.uiScale
	}
	if l.textScale > 0 {
		appearance.TextScale = l.textScale
	}
}

// ClearSizes clears CLI scale overrides when the user configures explicit values.
func (l *Scales) ClearSizes(current Appearance) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if current.InterfaceScale > 0 {
		l.uiScale = 0
	}
	if current.TextScale > 0 {
		l.textScale = 0
	}
}
