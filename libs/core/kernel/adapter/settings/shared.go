package settings

import "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"

// Shared holds the settings files and the change events that one composition root shares.
type Shared struct {
	Files  *Files
	Events *settingevents.Events
}

// NewShared returns settings files and change events with no reader.
func NewShared() Shared { return Shared{Files: NewFiles(), Events: settingevents.New()} }
