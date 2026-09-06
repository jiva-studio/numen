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
	// MaxNoteBytes is the most a note may be and still be read whole. Zero
	// means the default.
	MaxNoteBytes int64
}

// DefaultHold is how long events are held before they are acted on. One save
// is several events, and the same path arrives more than once inside it.
const DefaultHold = 50 * time.Millisecond

// ignoring is what a vault leaves out: the rules no vault has to ask for, and
// the rules the vault asked for beside them. A path either of them names is
// left out.
//
// They are two matchers and not one list because a vault is data. A vault's
// configuration is written by whoever synced it, and in one list the last
// pattern wins: `!.*` there would hand the application every dotfile folder in
// the vault to read from and write into. A vault narrows what is read and
// written and never widens it, so a negation it writes reaches only what the
// same vault asked to leave out.
type ignoring struct {
	defaults *ignore.GitIgnore
	vaults   *ignore.GitIgnore
}

// MatchesPath is asked for files and folders alike, against the path from the
// vault root, which is what the patterns are written in terms of.
func (i *ignoring) MatchesPath(path string) bool {
	return i.defaults.MatchesPath(path) || i.vaults.MatchesPath(path)
}

func (o Options) ignored() *ignoring {
	return &ignoring{
		defaults: ignore.CompileIgnoreLines(DefaultIgnore...),
		vaults:   ignore.CompileIgnoreLines(o.Ignore...),
	}
}

// DefaultMaxNoteBytes is the most a note is read whole at. It stands above
// every bound a caller holds its own reads to, and a file over it is a file
// this vault does not hold as a note.
const DefaultMaxNoteBytes = 16 << 20

func (o Options) maxNoteBytes() int64 {
	if o.MaxNoteBytes <= 0 {
		return DefaultMaxNoteBytes
	}
	return o.MaxNoteBytes
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

// isService says whether one component of a path is the folder the application
// keeps for itself. The name is compared without regard to case, which is how
// macOS and Windows open it, and which is how containment reads it.
func (o Options) isService(name string) bool {
	return strings.EqualFold(name, o.serviceDir())
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
