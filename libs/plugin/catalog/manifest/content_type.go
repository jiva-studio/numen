package manifest

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// ErrInvalidReader is a reader declared wrongly.
var ErrInvalidReader = errors.New("invalid reader")

// Reader is a tab of the plugin that opens files of the types it names.
type Reader struct {
	ID           string
	ContentTypes []domain.ContentType
	Rank         domain.HandlerRank
}

// ListContentHandlers lists the handler of each type each reader of the plugin opens.
func (m Manifest) ListContentHandlers() []domain.ContentHandler {
	var out []domain.ContentHandler
	for _, reader := range m.Contributes.Readers {
		for _, one := range reader.ContentTypes {
			out = append(out, domain.ContentHandler{PluginID: m.ID, ID: reader.ID, Type: one, Rank: reader.Rank})
		}
	}
	return out
}

// IsHandling reports whether a reader of the plugin takes a file of the type.
func (m Manifest) IsHandling(contentType domain.ContentType) bool {
	for _, reader := range m.Contributes.Readers {
		if isAnyMatching(reader.ContentTypes, contentType) {
			return true
		}
	}
	return false
}

func isAnyMatching(masks []domain.ContentType, contentType domain.ContentType) bool {
	for _, mask := range masks {
		if mask.Matches(contentType) {
			return true
		}
	}
	return false
}

func (m Manifest) listReaderIssues() []error {
	var issues []error
	if len(m.Contributes.Readers) > 0 && !m.HasFrontend() {
		issues = append(issues, fmt.Errorf("%w: the plugin declares readers and has no frontend to show them", ErrInvalidReader))
	}
	observed := map[string]bool{}
	for _, reader := range m.Contributes.Readers {
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
