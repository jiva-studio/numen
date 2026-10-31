package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrInvalidHandle is a handle declared wrongly.
var ErrInvalidHandle = errors.New("invalid handle")

var handlePurposePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// Handle declares that the plugin takes part in a capability for files of the content types
// ContentTypes names and for notes of its own types NoteTypes names. Purpose is the id of the
// capability. The part of the application that defines the capability owns the id and gives the
// declaration its meaning; the manifest reader checks only its shape.
type Handle struct {
	Purpose      string
	ContentTypes []domain.ContentType
	NoteTypes    []string
}

// IsMatching reports whether the handle covers files of the content type, or notes of the note
// type. An ordinary note has the type domain.TypeNote, which no plugin declares.
func (h Handle) IsMatching(contentType domain.ContentType, noteType domain.NoteType) bool {
	return isAnyMatching(h.ContentTypes, contentType) || slices.Contains(h.NoteTypes, string(noteType))
}

// HasNoteTypes reports whether the handle covers notes of some of the plugin's types.
func (h Handle) HasNoteTypes() bool { return len(h.NoteTypes) > 0 }

// GetHandle returns the handle of the purpose, and false where the plugin declares none.
func (m Manifest) GetHandle(purpose string) (Handle, bool) {
	for _, declared := range m.Contributes.Handles {
		if declared.Purpose == purpose {
			return declared, true
		}
	}
	return Handle{}, false
}

// listHandleIssues checks that each handle has a purpose declared once, that each content type is a
// type or a mask, and that each note type is one the plugin declares.
func (m Manifest) listHandleIssues() []error {
	var issues []error
	observed := map[string]bool{}
	for _, declared := range m.Contributes.Handles {
		switch {
		case !handlePurposePattern.MatchString(declared.Purpose):
			issues = append(issues, fmt.Errorf("%w: purpose %q is not lowercase words joined by hyphens", ErrInvalidHandle, declared.Purpose))
		case observed[declared.Purpose]:
			issues = append(issues, fmt.Errorf("%w: purpose %q is declared twice", ErrInvalidHandle, declared.Purpose))
		}
		observed[declared.Purpose] = true
		issues = append(issues, listContentTypesIssues(ErrInvalidHandle, declared.ContentTypes)...)
		issues = append(issues, m.listUndeclaredNoteTypes(ErrInvalidHandle, declared.NoteTypes)...)
	}
	return issues
}

// listUndeclaredNoteTypes lists, grouped by declaration kind, each note type
// that is not a type the plugin declares.
func (m Manifest) listUndeclaredNoteTypes(kind error, noteTypes []string) []error {
	var issues []error
	for _, noteType := range noteTypes {
		if _, ok := m.GetNoteType(noteType); !ok {
			issues = append(issues, fmt.Errorf("%w: %q is no note type the plugin declares", kind, noteType))
		}
	}
	return issues
}
