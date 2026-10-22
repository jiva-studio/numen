package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"

	"github.com/jiva-studio/numen/modules/libs/core/port/windowport"
)

// FocusPassage builds what opens a document in front of the person at a
// passage, through their window.
func (c Config) FocusPassage(view windowport.Window) *source.FocusPassage {
	return source.NewFocusPassage(c.VaultReaders(), view)
}
