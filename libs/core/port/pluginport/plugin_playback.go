package pluginport

import (
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrInvalidPlayback is a playback declaration without a content type or note type, or with an invalid one.
var ErrInvalidPlayback = errors.New("invalid playback")

// PluginPlayback declares that the plugin keeps a copy to play in place of files of the content
// types ContentTypes names and of notes of its own types NoteTypes names. The copy is a file of the
// plugin's store folder named by the content hash of the file and an extension of an audio or video
// type: `<hash>.mp4`.
type PluginPlayback struct {
	ContentTypes []domain.ContentType `json:"contentTypes,omitempty"`
	NoteTypes    []string             `json:"noteTypes,omitempty"`
}

// IsCopying reports whether the plugin keeps copies of files of the content type, or of notes of
// the note type. An ordinary note has the type domain.TypeNote, which no plugin declares.
func (p PluginPlayback) IsCopying(contentType domain.ContentType, noteType domain.NoteType) bool {
	return isAnyMatching(p.ContentTypes, contentType) || slices.Contains(p.NoteTypes, string(noteType))
}

// HasNoteTypes reports whether the plugin keeps copies of notes of some of its types.
func (p PluginPlayback) HasNoteTypes() bool { return len(p.NoteTypes) > 0 }

// listPlaybackIssues checks that a playback declaration names a content type or a note type, each
// content type a type or a mask and each note type one the plugin declares.
func (m PluginManifest) listPlaybackIssues() []error {
	in := m.Playback
	if in == nil {
		return nil
	}
	if len(in.ContentTypes) == 0 && len(in.NoteTypes) == 0 {
		return []error{fmt.Errorf("%w: it names no content type and no note type", ErrInvalidPlayback)}
	}
	issues := listContentTypesIssues(ErrInvalidPlayback, in.ContentTypes)
	return append(issues, m.listUndeclaredNoteTypes(ErrInvalidPlayback, in.NoteTypes)...)
}

// listUndeclaredNoteTypes names, under the kind of the declaration, each note type that is no type the
// plugin declares.
func (m PluginManifest) listUndeclaredNoteTypes(kind error, noteTypes []string) []error {
	var issues []error
	for _, noteType := range noteTypes {
		if _, ok := m.GetNoteType(noteType); !ok {
			issues = append(issues, fmt.Errorf("%w: %q is no note type the plugin declares", kind, noteType))
		}
	}
	return issues
}
