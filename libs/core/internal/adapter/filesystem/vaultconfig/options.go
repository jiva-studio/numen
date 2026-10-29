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
	// Ignore lists the patterns the vault excludes, in `.gitignore` syntax. Empty
	// means the default.
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

// Ignoring excludes paths from the vault: the default rules and the rules the
// vault configures. A path matched by either set is excluded.
//
// The two sets are separate matchers because vault configuration is untrusted
// data. In a single list the last pattern wins, so a vault pattern `!.*` would
// expose every dotfile folder to reads and writes. Keeping the sets separate
// means a vault can only narrow what is read and written, and a negation in
// vault rules affects only the vault's own exclusions.
type Ignoring struct {
	defaults *ignore.GitIgnore
	vaults   *ignore.GitIgnore
}

// MatchesPath reports whether the path is excluded. It applies to files and
// folders, and the path is relative to the vault root.
func (i *Ignoring) MatchesPath(path string) bool {
	return i.defaults.MatchesPath(path) || i.vaults.MatchesPath(path)
}

func (o Options) CompileIgnoring() *Ignoring {
	return &Ignoring{
		defaults: ignore.CompileIgnoreLines(DefaultIgnore...),
		vaults:   ignore.CompileIgnoreLines(o.Ignore...),
	}
}

// DefaultMaxNoteBytes is the maximum size at which a note is read whole. It
// exceeds every limit callers apply to their own reads, and a larger file is
// not treated as a note by this vault.
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

// IsService reports whether one component of a path is the application's
// service folder. The name is compared case-insensitively, matching how macOS
// and Windows open it and how containment checks resolve it.
func (o Options) IsService(name string) bool {
	return strings.EqualFold(name, o.GetServiceDir())
}

// Kind returns the source kind of a file name: a note or any other file.
func (o Options) Kind(name string) domain.SourceKind {
	if hasExtension(name, NoteExtensions) {
		return domain.KindNote
	}
	return domain.KindFile
}

// IsNote reports whether a file of this name is a note. The application writes
// notes into the vault folder; every other file there is user content.
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
