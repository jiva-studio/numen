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
// ContentTypes names, and text versions of the kinds Kinds names. IsAutoRun
// asks the host to run the plugin on a file of its types that has no live text
// version.
//
// A plugin that declares one serves numen.v1.TextProviderService.
type PluginTextInput struct {
	ContentTypes []domain.ContentType `json:"contentTypes,omitempty"`
	Kinds        []domain.TextKind    `json:"kinds,omitempty"`
	IsAutoRun    bool                 `json:"autoRun,omitempty"`
}

// listTextInputIssues checks that a text input names something to read, every
// content type as a type or a mask and every kind as a non-empty name.
func (m PluginManifest) listTextInputIssues() []error {
	in := m.TextInput
	if in == nil {
		return nil
	}
	var issues []error
	if len(in.ContentTypes) == 0 && len(in.Kinds) == 0 {
		issues = append(issues, fmt.Errorf("%w: it names no content type and no kind", ErrInvalidTextInput))
	}
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

// PluginTexts is the text versions of the vault the host shows, as the plugins
// that call the host work them.
type PluginTexts interface {
	// PutVersion records a version of the plugin, or replaces the one it has under the key.
	PutVersion(ctx context.Context, pluginID string, params PluginTextPut) (domain.TextVersion, error)
	// RemoveVersion removes the record of a version of the plugin.
	RemoveVersion(ctx context.Context, pluginID, path, key string) error
	// ListVersions is the versions of a file and which of them is live.
	ListVersions(ctx context.Context, path string) (PluginTextListing, error)
	// ReadText is one page of any version's segments, read from its file.
	ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (PluginTextPage, error)
}

// PluginTextHost receives the text versions that plugins work on.
type PluginTextHost interface {
	// SetPluginTexts is the text versions of the shown vault the plugins work. Nil is none.
	SetPluginTexts(texts PluginTexts)
}
