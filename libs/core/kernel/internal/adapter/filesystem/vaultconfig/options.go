package vaultconfig

import (
	"time"

	ignore "github.com/sabhiram/go-gitignore"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Options are what a vault on disk may be configured with.
type Options struct {
	ServiceDir    string
	Ignore        []string
	DebounceDelay time.Duration
	MaxNoteBytes  int64
}

// DefaultDebounceDelay is how long events are held before they are acted on. One save
// is several events, and the same path arrives more than once inside it.
const DefaultDebounceDelay = 50 * time.Millisecond

// Ignoring excludes paths from the vault: the default rules and the rules the
// vault configures. A path matched by either set is excluded.
// The sets are separate matchers because vault configuration is untrusted: in
// one list the last pattern wins, so a vault pattern `!.*` would expose every
// dotfile folder. Separate sets let a vault only narrow what is read and written.
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
		defaults: ignore.CompileIgnoreLines(vaultdomain.GetDefaultIgnore()...),
		vaults:   ignore.CompileIgnoreLines(o.Ignore...),
	}
}

func (o Options) GetMaxNoteBytes() int64 {
	if o.MaxNoteBytes <= 0 {
		return vaultdomain.DefaultMaxNoteBytes
	}
	return o.MaxNoteBytes
}

func (o Options) GetDebounceDelay() time.Duration {
	if o.DebounceDelay <= 0 {
		return DefaultDebounceDelay
	}
	return o.DebounceDelay
}

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
	return vaultdomain.IsServiceDir(name, o.GetServiceDir())
}

// Kind returns the source kind for a file name: a note or any other file.
func (o Options) Kind(name string) domain.SourceKind {
	return vaultdomain.GetSourceKind(name)
}

// IsNote reports whether a file of this name is a note. The application writes
// notes into the vault folder; every other file there is user content.
func (o Options) IsNote(name string) bool {
	return vaultdomain.IsNote(name)
}
