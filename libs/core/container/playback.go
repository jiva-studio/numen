package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/playback"
	"github.com/jiva-studio/numen/libs/core/port/playbackport"
)

// Copies are the copies the enabled plugins keep of the vault's files in their store folders. An
// installation with no plugins keeps none.
func (c Config) Copies() playbackport.Copies {
	if c.PluginHost == nil || c.SafeMode {
		return noCopies{}
	}
	return playback.NewCopies(c.PluginHost, derivedfile.PluginStores{Options: c.vaultOptions()})
}

// noCopies is an installation where no plugin keeps a copy.
type noCopies struct{}

func (noCopies) IsCopied(string) bool { return false }

func (noCopies) OpenCopy(context.Context, domain.Vault, string, string) (playbackport.Copy, error) {
	return playbackport.Copy{}, playbackport.ErrNoCopy
}
