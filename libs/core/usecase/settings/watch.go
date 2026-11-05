package settings

import "github.com/jiva-studio/numen/libs/core/port/settingsport"

// Watch follows the settings that change, whoever changes them: a write by a window, a plugin or
// the palette, and an edit of the file by hand.
type Watch struct {
	subscribe settingsport.ChangeSubscriber
}

func NewWatch(subscribe settingsport.ChangeSubscriber) *Watch {
	return &Watch{subscribe: subscribe}
}

// Execute starts listening. Each batch holds the settings that changed together, named by their
// declared path, once. Calling the returned function stops the delivery.
func (u *Watch) Execute() (<-chan []Setting, func()) {
	return u.subscribe()
}
