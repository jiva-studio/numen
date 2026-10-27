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
	// ErrNotTextPlugin is a plugin that is not enabled or declares no text input.
	ErrNotTextPlugin = errors.New("the plugin makes no text")
	// ErrInvalidVersion is a version with no key, no kind or no file, a file
	// that is not there or is not JSON Lines of segments, or a source that is
	// no other version of the file.
	ErrInvalidVersion = errors.New("the version is not valid")
	// ErrSourceNotAccepted is a source the plugin's declaration does not take.
	ErrSourceNotAccepted = errors.New("the plugin does not take this source")
)

// TextCreation asks a plugin for text of a file. Source is the version it is
// made from, and empty for the file itself. IsUserStarted is set for text a
// person asked for, which runs before text nobody asked for.
type TextCreation struct {
	Vault         domain.VaultID
	Path          string
	Source        domain.TextVersionID
	IsUserStarted bool
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

// TextPlugin is an enabled plugin that makes text, and what it makes text
// from: files of the types Files names, versions of the kinds Kinds names, and
// notes of the plugin's own types NoteTypes names. IsAutoRun asks for the
// plugin to be run on a file of its types that has no live version. Name is
// the plugin's name as a person reads it. Menu is the plugin's entry in the
// menu of a file its input takes, and has no title for a plugin not offered
// there.
type TextPlugin struct {
	ID        string
	Name      string
	Files     domain.FileInput
	Kinds     []domain.TextKind
	NoteTypes []domain.NoteType
	IsAutoRun bool
	Menu      MenuItem
}

// MenuItem is what a plugin's entry in a file's menu reads and is drawn with.
// Icon is a Lucide icon name.
type MenuItem struct {
	Title string
	Icon  string
}

// TextPlugins is the plugins that make text.
type TextPlugins interface {
	// GetTextPlugins is every enabled plugin that declares a text input, in the order of their ids.
	GetTextPlugins() []TextPlugin
}
