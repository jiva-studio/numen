package pluginport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrInvalidTextInput is a text input that names nothing to read, or names it wrongly.
var ErrInvalidTextInput = errors.New("invalid text input")

// PluginTextInput is what a plugin makes text from: files of the types Files
// names, by extension with its dot, and text versions in the formats Formats
// names. IsAutoRun asks the host to run the plugin on a file of its types that
// has no live text version.
//
// A plugin that declares one serves numen.v1.TextProviderService.
type PluginTextInput struct {
	Files     []string            `json:"files,omitempty"`
	Formats   []domain.TextFormat `json:"formats,omitempty"`
	IsAutoRun bool                `json:"autoRun,omitempty"`
}

// listTextInputIssues checks that a text input names something to read, every
// file type as an extension and every format as one the core reads.
func (m PluginManifest) listTextInputIssues() []error {
	in := m.TextInput
	if in == nil {
		return nil
	}
	var issues []error
	if len(in.Files) == 0 && len(in.Formats) == 0 {
		issues = append(issues, fmt.Errorf("%w: it names no file type and no format", ErrInvalidTextInput))
	}
	for _, ext := range in.Files {
		if len(ext) < 2 || !strings.HasPrefix(ext, ".") || strings.ContainsAny(ext[1:], `./\ `) {
			issues = append(issues, fmt.Errorf("%w: %q is not an extension", ErrInvalidTextInput, ext))
		}
	}
	for _, format := range in.Formats {
		if !format.IsValid() {
			issues = append(issues, fmt.Errorf("%w: %q is not a format, expected one of %v", ErrInvalidTextInput, format, domain.TextFormats()))
		}
	}
	if in.IsAutoRun && len(in.Files) == 0 {
		issues = append(issues, fmt.Errorf("%w: a plugin that runs on its own reads files", ErrInvalidTextInput))
	}
	return issues
}

// PluginTextPut is what a plugin reports of one of its text versions.
type PluginTextPut struct {
	Path      string
	Key       string
	Input     domain.TextVersionID
	Format    domain.TextFormat
	InputHash string
	Hash      string
}

// PluginTextListing is the versions of a file and the live one, which is empty
// where none is live.
type PluginTextListing struct {
	Versions []domain.TextVersion
	Live     domain.TextVersionID
}

// PluginTextPage is one page of a version's text. An empty NextCursor is the last page.
type PluginTextPage struct {
	Parts      []domain.TextPart
	NextCursor string
}

// PluginTexts is the text versions of the vault the host shows, as the plugins
// that call the host work them.
type PluginTexts interface {
	// PutVersion records a version of the plugin, or replaces the one it has under the key.
	PutVersion(ctx context.Context, pluginID string, put PluginTextPut) (domain.TextVersion, error)
	// RemoveVersion removes the record of a version of the plugin.
	RemoveVersion(ctx context.Context, pluginID, path, key string) error
	// ListVersions is the versions of a file and which of them is live.
	ListVersions(ctx context.Context, path string) (PluginTextListing, error)
	// ReadText is one page of any version's text, read from the plugin that made it.
	ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (PluginTextPage, error)
}

// PluginTextHost is what the host is told about the text versions plugins work.
type PluginTextHost interface {
	// SetPluginTexts is the text versions of the shown vault the plugins work. Nil is none.
	SetPluginTexts(texts PluginTexts)
	// SetOnEnabledChanged registers a callback invoked with the id of a plugin the person enabled or disabled.
	SetOnEnabledChanged(fn func(pluginID string))
}
