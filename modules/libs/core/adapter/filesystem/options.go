package filesystem

import (
	"strings"
	"time"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Options are what a vault on disk may be configured with.
type Options struct {
	// ServiceDir is the folder the application keeps its own files in. Empty
	// means the default.
	ServiceDir string
	// BookExtensions are the file extensions treated as books, with the leading
	// dot. Empty means the default, which is EPUB and PDF.
	BookExtensions []string
	// RecordingExtensions are the file extensions treated as recordings, with
	// the leading dot. Empty means the default.
	RecordingExtensions []string
	// Ignore is what the vault says not to look at, in the syntax of
	// `.gitignore`. Empty means the default.
	Ignore []string
	// Hold is how long changes are kept before they are reported. Zero means
	// the default.
	Hold time.Duration
}

// DefaultHold is how long events are held before they are acted on. One save
// is several events, and the same path arrives more than once inside it.
const DefaultHold = 50 * time.Millisecond

// ignored answers for files and folders alike, against the path from the vault
// root, which is what the patterns are written in terms of.
//
// What a vault says is added to the defaults: a vault asking for its archive
// to be left alone is not asking for an editor's lock files to be indexed.
func (o Options) ignored() *ignore.GitIgnore {
	return ignore.CompileIgnoreLines(append(append([]string(nil), DefaultIgnore...), o.Ignore...)...)
}

func (o Options) hold() time.Duration {
	if o.Hold <= 0 {
		return DefaultHold
	}
	return o.Hold
}

// NoteExtensions is what counts as a note. A note, a deck, a stencil and a
// preset are all markdown, and a vault holds them under one extension.
var NoteExtensions = []string{domain.NoteExtension}

// DefaultBookExtensions is what counts as a book: the formats a reader takes
// text out of. A book is any source with text that a person did not type here.
var DefaultBookExtensions = []string{".epub", ".pdf"}

// DefaultRecordingExtensions is what counts as a recording: the containers a
// model is given speech out of. A recording is a source whose words nobody has
// written down yet.
var DefaultRecordingExtensions = domain.RecordingExtensions()

// DefaultIgnore is what no vault has to ask to be left out. A name beginning
// with a dot belongs to a tool — an editor's lock, a sync client's
// bookkeeping.
var DefaultIgnore = []string{".*"}

func (o Options) serviceDir() string {
	if o.ServiceDir == "" {
		return DefaultServiceDir
	}
	return o.ServiceDir
}

func (o Options) bookExtensions() []string {
	if len(o.BookExtensions) == 0 {
		return DefaultBookExtensions
	}
	return o.BookExtensions
}

// kind says which sort of source a file's name makes it, and whether it is one
// at all. A name that answers to more than one list is a note.
func (o Options) kind(name string) (domain.SourceKind, bool) {
	switch {
	case named(name, NoteExtensions):
		return domain.KindNote, true
	case named(name, o.bookExtensions()):
		return domain.KindBook, true
	case named(name, o.recordingExtensions()):
		return domain.KindRecording, true
	}
	return "", false
}

func (o Options) recordingExtensions() []string {
	if len(o.RecordingExtensions) == 0 {
		return DefaultRecordingExtensions
	}
	return o.RecordingExtensions
}

func (o Options) isNote(name string) bool {
	kind, ok := o.kind(name)
	return ok && kind == domain.KindNote
}

func named(name string, extensions []string) bool {
	for _, ext := range extensions {
		if len(name) > len(ext) && strings.EqualFold(name[len(name)-len(ext):], ext) {
			return true
		}
	}
	return false
}
