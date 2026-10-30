package manifest

import (
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrInvalidTexts is a `texts` declaration that names a content type wrongly.
var ErrInvalidTexts = errors.New("invalid texts")

// TextMaker is what a plugin declares of the text it makes: it puts text
// versions of vault files. ContentTypes, where it names any, are the types of
// the files whose text the host requests from the plugin when they have no live
// text version; the plugin then serves numen.v1.TextProviderService. A plugin
// that makes its text when a command runs names none.
type TextMaker struct {
	ContentTypes []domain.ContentType
}

// listTextsIssues checks that every content type of the declaration is a type or a mask.
func (m Manifest) listTextsIssues() []error {
	if m.Texts == nil {
		return nil
	}
	return listContentTypesIssues(ErrInvalidTexts, m.Texts.ContentTypes)
}
