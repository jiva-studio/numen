package settings

import "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"

// Shared holds the settings files and the change events that one composition root shares.
type Shared struct {
	Files  *Files
	Events *settingevents.Events
}

// NewShared returns the settings files and a feed of their changes, shared by one composition root.
func NewShared() Shared {
	return Shared{Files: NewFiles(), Events: settingevents.New()}
}
