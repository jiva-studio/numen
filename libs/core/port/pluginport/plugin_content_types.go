package pluginport

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrInvalidReader is a reader declared wrongly.
var ErrInvalidReader = errors.New("invalid reader")

// PluginContentType is a type of file the plugin brings, and the names of the files of it.
type PluginContentType struct {
	Type domain.ContentType
	// Globs are the patterns of the names, each `*.` and an extension: `*.sample`.
	Globs []string
}

// ListExtensions lists the extensions of the globs, lowercase and without the dot.
func (t PluginContentType) ListExtensions() []string {
	out := make([]string, 0, len(t.Globs))
	for _, glob := range t.Globs {
		out = append(out, strings.ToLower(strings.TrimPrefix(glob, "*.")))
	}
	return out
}

// PluginReader is a tab of the plugin that opens files of the types it names.
type PluginReader struct {
	// ID is the id the frontend registers the reader under.
	ID           string
	ContentTypes []domain.ContentType
	Rank         domain.HandlerRank
}

// ListContentHandlers lists the handler of each type each reader of the plugin opens.
func (m PluginManifest) ListContentHandlers() []domain.ContentHandler {
	var out []domain.ContentHandler
	for _, reader := range m.Readers {
		for _, one := range reader.ContentTypes {
			out = append(out, domain.ContentHandler{PluginID: m.ID, ID: reader.ID, Type: one, Rank: reader.Rank})
		}
	}
	return out
}

// IsHandling reports whether a reader or the text input of the plugin takes a file of the type.
func (m PluginManifest) IsHandling(contentType domain.ContentType) bool {
	for _, reader := range m.Readers {
		if isAnyMatching(reader.ContentTypes, contentType) {
			return true
		}
	}
	return m.TextInput != nil && isAnyMatching(m.TextInput.ContentTypes, contentType)
}

func isAnyMatching(masks []domain.ContentType, contentType domain.ContentType) bool {
	for _, mask := range masks {
		if mask.Matches(contentType) {
			return true
		}
	}
	return false
}

// listContentTypeIssues checks that each type the plugin brings is one type, declared once, with
// globs of extensions.
func (m PluginManifest) listContentTypeIssues() []error {
	var issues []error
	observed := map[domain.ContentType]bool{}
	for _, declared := range m.ContentTypes {
		if err := checkContentType(declared.Type); err != nil {
			issues = append(issues, err)
		}
		switch {
		case declared.Type.IsMask() || declared.Type == domain.AnyContentType:
			issues = append(issues, fmt.Errorf("%w: %q is a mask, and a plugin brings types", domain.ErrInvalidContentType, declared.Type))
		case observed[declared.Type]:
			issues = append(issues, fmt.Errorf("%w: %q is declared twice", domain.ErrInvalidContentType, declared.Type))
		case len(declared.Globs) == 0:
			issues = append(issues, fmt.Errorf("%w: %q has no globs", domain.ErrInvalidContentType, declared.Type))
		}
		observed[declared.Type] = true
		for _, glob := range declared.Globs {
			extension, ok := strings.CutPrefix(glob, "*.")
			if !ok || extension == "" || extension != strings.ToLower(extension) || strings.ContainsAny(extension, `.*?[/\ `) {
				issues = append(issues, fmt.Errorf("%w: %q has glob %q, which is not *. and a lowercase extension", domain.ErrInvalidContentType, declared.Type, glob))
			}
		}
	}
	return issues
}

// listReaderIssues checks that each reader has an id of its own, a rank and types, and that the
// plugin has a frontend to show it.
func (m PluginManifest) listReaderIssues() []error {
	var issues []error
	if len(m.Readers) > 0 && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares readers and has no frontend to show them", ErrInvalidReader))
	}
	observed := map[string]bool{}
	for _, reader := range m.Readers {
		switch {
		case strings.TrimSpace(reader.ID) == "":
			issues = append(issues, fmt.Errorf("%w: a reader has no id", ErrInvalidReader))
		case observed[reader.ID]:
			issues = append(issues, fmt.Errorf("%w: %q is declared twice", ErrInvalidReader, reader.ID))
		}
		observed[reader.ID] = true
		if !reader.Rank.IsValid() {
			issues = append(issues, fmt.Errorf("%w: %q has rank %q, expected %q or %q", ErrInvalidReader, reader.ID, reader.Rank, domain.RankDefault, domain.RankAlternate))
		}
		if len(reader.ContentTypes) == 0 {
			issues = append(issues, fmt.Errorf("%w: %q opens no content type", ErrInvalidReader, reader.ID))
		}
		issues = append(issues, listContentTypesIssues(ErrInvalidReader, reader.ContentTypes)...)
	}
	return issues
}

// listContentTypesIssues checks that each type or mask is lowercase, and
// returns an issue of the given kind for each one that is not.
func listContentTypesIssues(kind error, types []domain.ContentType) []error {
	var issues []error
	for _, one := range types {
		if err := checkContentType(one); err != nil {
			issues = append(issues, fmt.Errorf("%w: %w", kind, err))
		}
	}
	return issues
}

func checkContentType(contentType domain.ContentType) error {
	parsed, err := domain.ParseContentType(string(contentType))
	if err != nil {
		return err
	}
	if parsed != contentType {
		return fmt.Errorf("%w: %q is not written in lowercase", domain.ErrInvalidContentType, contentType)
	}
	return nil
}
