package vault

import (
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// DefaultMaxNoteBytes is the maximum size at which a note is read whole. It
// exceeds every limit callers apply to their own reads, and a larger file is
// not treated as a note by this vault.
const DefaultMaxNoteBytes = 16 << 20

// GetNoteExtensions returns what counts as a note. A note of every type a plugin
// declares is markdown, and a vault holds them under one extension.
func GetNoteExtensions() []string { return []string{domain.NoteExtension} }

// GetDefaultIgnore returns what no vault has to ask to be left out. A name beginning
// with a dot belongs to a tool — an editor's lock, a sync client's
// bookkeeping.
func GetDefaultIgnore() []string { return []string{".*"} }

// IsServiceDir reports whether one component of a path is the application's
// service folder. The name is compared case-insensitively, matching how macOS
// and Windows open it and how containment checks resolve it.
func IsServiceDir(name, serviceDir string) bool {
	return strings.EqualFold(name, serviceDir)
}

// GetSourceKind returns the source kind for a file name: a note or any other file.
func GetSourceKind(name string) domain.SourceKind {
	if hasExtension(name, GetNoteExtensions()) {
		return domain.KindNote
	}
	return domain.KindFile
}

// IsNote reports whether a file of this name is a note. The application writes
// notes into the vault folder; every other file there is user content.
func IsNote(name string) bool {
	return GetSourceKind(name) == domain.KindNote
}

func hasExtension(name string, extensions []string) bool {
	for _, ext := range extensions {
		if len(name) > len(ext) && strings.EqualFold(name[len(name)-len(ext):], ext) {
			return true
		}
	}
	return false
}
