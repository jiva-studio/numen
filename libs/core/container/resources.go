package container

import (
	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
)

// Resources is what every part a container assembles shares: the write lock of each vault, the
// folder watches and the access to the settings file. An application builds one with
// NewResources and names it in its Config.
type Resources struct {
	vaultLocks    *vaultwriter.Locks
	watches       *vaultwatch.Hub
	settingsFiles *settings.Files
}

// NewResources returns the resources of one container.
func NewResources() *Resources {
	return &Resources{
		vaultLocks:    vaultwriter.NewLocks(),
		watches:       vaultwatch.NewHub(),
		settingsFiles: settings.NewFiles(),
	}
}

// NewConfig returns c with the resources of one container. A Config built without NewConfig and
// without Resources opens writers, watches and settings that refuse to work.
func NewConfig(c Config) Config {
	if c.Resources == nil {
		c.Resources = NewResources()
	}
	return c
}

// getResources returns the resources the Config names, or empty ones whose locks, watches and
// settings access return an error.
func (c Config) getResources() *Resources {
	if c.Resources != nil {
		return c.Resources
	}
	return &Resources{vaultLocks: nil, watches: nil, settingsFiles: nil}
}
