package pluginport

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrInvalidTexts is a `texts` declaration that names a content type wrongly.
var ErrInvalidTexts = errors.New("invalid texts")

// PluginTextMaker is what a plugin declares of the text it makes: it puts text versions of vault
// files. ContentTypes, where it names any, are the types of the files the host asks the plugin
// for the text of when they have no live text version; the plugin then serves
// numen.v1.TextProviderService. A plugin that makes its text when a command runs names none.
type PluginTextMaker struct {
	ContentTypes []domain.ContentType
}

// listTextsIssues checks that every content type of the declaration is a type or a mask.
func (m PluginManifest) listTextsIssues() []error {
	if m.Texts == nil {
		return nil
	}
	return listContentTypesIssues(ErrInvalidTexts, m.Texts.ContentTypes)
}

// PluginTextPut is what a plugin reports of one of its text versions. File is
// the name of the version's file in the plugin's folder.
type PluginTextPut struct {
	Path       string
	Key        string
	Source     domain.TextVersionID
	Kind       domain.TextKind
	SourceHash string
	File       string
}

// PluginTextListing is the versions of a file and the live one, which is empty
// where none is live.
type PluginTextListing struct {
	Versions []domain.TextVersion
	Live     domain.TextVersionID
}

// PluginTextPage is one page of a version's segments. An empty NextCursor is the last page.
type PluginTextPage struct {
	Segments   []domain.TextSegment
	NextCursor string
}

// PluginTexts is the text versions of the vaults, as the plugins that call the
// host work them. Each call works the vault it is given.
type PluginTexts interface {
	// PutVersion records a version of the plugin, or replaces the one it has under the key.
	PutVersion(ctx context.Context, v domain.Vault, pluginID string, params PluginTextPut) (domain.TextVersion, error)
	// RemoveVersion removes the record of a version of the plugin.
	RemoveVersion(ctx context.Context, v domain.Vault, pluginID, path, key string) error
	// ListVersions is the versions of a file and which of them is live.
	ListVersions(ctx context.Context, v domain.Vault, path string) (PluginTextListing, error)
	// ReadText is one page of any version's segments, read from its file.
	ReadText(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, cursor string) (PluginTextPage, error)
}

// PluginTextHost receives the text versions that plugins work on.
type PluginTextHost interface {
	// SetPluginTexts is the text versions the plugins work. Nil is none.
	SetPluginTexts(texts PluginTexts)
}
