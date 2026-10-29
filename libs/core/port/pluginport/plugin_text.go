package pluginport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrInvalidTextInput is a text input that points at no version, or at the wrong one.
var ErrInvalidTextInput = errors.New("invalid text input")

// PluginTextInput is what a plugin makes text from: files of the content types
// ContentTypes names, text versions of the kinds Kinds names, and notes of the
// plugin's own types NoteTypes names. IsAutoRun asks the host to run the plugin
// on a file of its content types that has no live text version. The plugin
// starts the text of a note of its types itself.
//
// A plugin that declares one serves numen.v1.TextProviderService.
type PluginTextInput struct {
	ContentTypes []domain.ContentType
	Kinds        []domain.TextKind
	NoteTypes    []string
	IsAutoRun    bool
}

// listTextInputIssues checks that a text input names something to read, every
// content type as a type or a mask, every kind as a non-empty name and every note
// type as one the plugin declares.
func (m PluginManifest) listTextInputIssues() []error {
	in := m.TextInput
	if in == nil {
		return nil
	}
	var issues []error
	if len(in.ContentTypes) == 0 && len(in.Kinds) == 0 && len(in.NoteTypes) == 0 {
		issues = append(issues, fmt.Errorf("%w: it names no content type, no kind and no note type", ErrInvalidTextInput))
	}
	issues = append(issues, m.listUndeclaredNoteTypes(ErrInvalidTextInput, in.NoteTypes)...)
	issues = append(issues, listContentTypesIssues(ErrInvalidTextInput, in.ContentTypes)...)
	for _, kind := range in.Kinds {
		if strings.TrimSpace(string(kind)) == "" {
			issues = append(issues, fmt.Errorf("%w: a kind has no name", ErrInvalidTextInput))
		}
	}
	if in.IsAutoRun && len(in.ContentTypes) == 0 {
		issues = append(issues, fmt.Errorf("%w: a plugin that runs on its own reads files", ErrInvalidTextInput))
	}
	return issues
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
