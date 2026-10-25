package textport

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

var (
	// ErrNoText is a file, a version or a key nothing has text for.
	ErrNoText = errors.New("there is no such text")
	// ErrNotTextPlugin is a plugin that is not enabled or declares no text input.
	ErrNotTextPlugin = errors.New("the plugin makes no text")
	// ErrInvalidVersion is a version with no key, a format the core does not
	// read, or an input that is no other version of the file.
	ErrInvalidVersion = errors.New("the version is not valid")
	// ErrInputNotAccepted is an input the plugin's declaration does not take.
	ErrInputNotAccepted = errors.New("the plugin does not take this input")
)

// TextRequest names one page of a text a plugin keeps. Cursor is the
// NextCursor of the page before, and empty for the first.
type TextRequest struct {
	Vault  domain.VaultID
	Path   string
	Key    string
	Cursor string
}

// TextPage is one page of a text. An empty NextCursor is the last page.
type TextPage struct {
	Parts      []domain.TextPart
	NextCursor string
}

// TextCreation asks a plugin for text of a file. Input is the version it is
// made from, and empty for the file itself. IsUserStarted is set for text a
// person asked for, which runs before text nobody asked for.
type TextCreation struct {
	Vault         domain.VaultID
	Path          string
	Input         domain.TextVersionID
	IsUserStarted bool
}

// TextProvider is a plugin that makes and keeps text of vault files.
type TextProvider interface {
	// GetText is one page of a text the plugin keeps, or ErrNoText.
	GetText(ctx context.Context, req TextRequest) (TextPage, error)
	// CreateText starts making text. The plugin puts the version once it has
	// text, which may be after the call returns.
	CreateText(ctx context.Context, req TextCreation) error
	// DeleteText removes a text the plugin keeps. A text that is not there is
	// removed.
	DeleteText(ctx context.Context, v domain.VaultID, path, key string) error
}

// TextProviders reaches the provider of an enabled plugin.
type TextProviders interface {
	OpenProvider(ctx context.Context, pluginID string) (TextProvider, error)
}

// TextPlugin is an enabled plugin that makes text, and what it makes text
// from: files of the types Files names, and versions in the formats Formats
// names. IsAutoRun asks for the plugin to be run on a file of its types that
// has no live version. Name is the plugin's name as a person reads it.
type TextPlugin struct {
	ID        string
	Name      string
	Files     domain.FileInput
	Formats   []domain.TextFormat
	IsAutoRun bool
}

// TextPlugins is the plugins that make text.
type TextPlugins interface {
	// GetTextPlugins is every enabled plugin that declares a text input, in the order of their ids.
	GetTextPlugins() []TextPlugin
}
