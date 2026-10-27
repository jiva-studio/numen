package vaultconfig

import (
	"strings"
	"time"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Options are what a vault on disk may be configured with.
type Options struct {
	// ServiceDir is the folder the application keeps its own files in. Empty
	// means the default.
	ServiceDir string
	// Ignore is what the vault says not to look at, in the syntax of
	// `.gitignore`. Empty means the default.
	Ignore []string
	// DebounceDelay is how long changes are kept before they are reported. Zero means
	// the default.
	DebounceDelay time.Duration
	// MaxNoteBytes is the most a note may be and still be read whole. Zero
	// means the default.
	MaxNoteBytes int64
}

// DefaultDebounceDelay is how long events are held before they are acted on. One save
// is several events, and the same path arrives more than once inside it.
const DefaultDebounceDelay = 50 * time.Millisecond

// Ignoring is what a vault leaves out: the rules no vault has to ask for, and
// the rules the vault asked for beside them. A path either of them names is
// left out.
//
// They are two matchers and not one list because a vault is data. A vault's
// configuration is written by whoever synced it, and in one list the last
// pattern wins: `!.*` there would hand the application every dotfile folder in
// the vault to read from and write into. A vault narrows what is read and
// written and never widens it, so a negation it writes reaches only what the
// same vault asked to leave out.
type Ignoring struct {
	defaults *ignore.GitIgnore
	vaults   *ignore.GitIgnore
}

// MatchesPath is asked for files and folders alike, against the path from the
// vault root, which is what the patterns are written in terms of.
func (i *Ignoring) MatchesPath(path string) bool {
	return i.defaults.MatchesPath(path) || i.vaults.MatchesPath(path)
}

func (o Options) CompileIgnoring() *Ignoring {
	return &Ignoring{
		defaults: ignore.CompileIgnoreLines(DefaultIgnore...),
		vaults:   ignore.CompileIgnoreLines(o.Ignore...),
	}
}

// DefaultMaxNoteBytes is the most a note is read whole at. It stands above
// every bound a caller holds its own reads to, and a file over it is a file
// this vault does not hold as a note.
const DefaultMaxNoteBytes = 16 << 20

func (o Options) GetMaxNoteBytes() int64 {
	if o.MaxNoteBytes <= 0 {
		return DefaultMaxNoteBytes
	}
	return o.MaxNoteBytes
}

func (o Options) GetDebounceDelay() time.Duration {
	if o.DebounceDelay <= 0 {
		return DefaultDebounceDelay
	}
	return o.DebounceDelay
}

// NoteExtensions is what counts as a note. A note of every type a plugin
// declares is markdown, and a vault holds them under one extension.
var NoteExtensions = []string{domain.NoteExtension}

// DefaultIgnore is what no vault has to ask to be left out. A name beginning
// with a dot belongs to a tool — an editor's lock, a sync client's
// bookkeeping.
var DefaultIgnore = []string{".*"}

func (o Options) GetServiceDir() string {
	if o.ServiceDir == "" {
		return DefaultServiceDir
	}
	return o.ServiceDir
}

// IsService says whether one component of a path is the folder the application
// keeps for itself. The name is compared without regard to case, which is how
// macOS and Windows open it, and which is how containment reads it.
func (o Options) IsService(name string) bool {
	return strings.EqualFold(name, o.GetServiceDir())
}

// Kind says which sort of source a file's name makes it: a note or any other file.
func (o Options) Kind(name string) domain.SourceKind {
	if hasExtension(name, NoteExtensions) {
		return domain.KindNote
	}
	return domain.KindFile
}

// IsNote says whether a file of this name is a note. The application writes
// notes into the person's folder; every other file there is somebody else's.
func (o Options) IsNote(name string) bool {
	return o.Kind(name) == domain.KindNote
}

func hasExtension(name string, extensions []string) bool {
	for _, ext := range extensions {
		if len(name) > len(ext) && strings.EqualFold(name[len(name)-len(ext):], ext) {
			return true
		}
	}
	return false
}
