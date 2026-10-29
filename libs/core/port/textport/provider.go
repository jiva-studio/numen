package textport

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

var (
	// ErrNoText is a file, a version or a key nothing has text for, or a
	// version whose file changed since it was recorded.
	ErrNoText = errors.New("there is no such text")
	// ErrNotTextPlugin is a plugin that is not enabled or declares no `texts`.
	ErrNotTextPlugin = errors.New("the plugin makes no text")
	// ErrInvalidVersion is a version with no key, no kind or no file, a file
	// that is not there or is not JSON Lines of segments, or a source that is
	// no other version of the file.
	ErrInvalidVersion = errors.New("the version is not valid")
	// ErrSourceNotAccepted is a source the plugin does not take.
	ErrSourceNotAccepted = errors.New("the plugin does not take this source")
)

// TextCreation asks a plugin for text of a file.
type TextCreation struct {
	Vault domain.VaultID
	Path  string
}

// TextProvider is a plugin that makes text of vault files.
type TextProvider interface {
	// CreateText starts making text. The plugin writes the version's file and
	// puts the version once it has text, which may be after the call returns.
	CreateText(ctx context.Context, req TextCreation) error
}

// TextProviders reaches the provider of an enabled plugin.
type TextProviders interface {
	OpenProvider(ctx context.Context, pluginID string) (TextProvider, error)
}

// TextPlugin is an enabled plugin that makes text. Files are the extensions of the files the host
// asks it for the text of when they have no live text version, and are empty for a plugin that
// makes its text when a command runs. Name is the plugin's name as a person reads it.
type TextPlugin struct {
	ID    string
	Name  string
	Files domain.FileInput
}

// TextPlugins is the plugins that make text.
type TextPlugins interface {
	// GetTextPlugins is every enabled plugin that declares a text input, in the order of their ids.
	GetTextPlugins() []TextPlugin
}
