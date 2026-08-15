package filesystem

import (
	"strings"
	"time"

	ignore "github.com/sabhiram/go-gitignore"
)

// Options are what a vault on disk may be configured with.
type Options struct {
	// ServiceDir is the folder the application keeps its own files in. Empty
	// means the default.
	ServiceDir string
	// Extensions are the file extensions treated as notes, with the leading
	// dot. Empty means the default, which is markdown alone.
	Extensions []string
	// Ignore is what the vault says not to look at, in the syntax of
	// `.gitignore`. Empty means the default.
	Ignore []string
	// Window is how long changes are collected before they are reported. Zero
	// means the default.
	Window time.Duration
}

// DefaultWindow is how long events are held before they are acted on. One save
// is several events, and the same path arrives more than once inside it.
const DefaultWindow = 50 * time.Millisecond

// ignored answers for files and folders alike, against the path from the vault
// root, which is what the patterns are written in terms of.
//
// What a vault says is added to the defaults rather than put in their place: a
// vault asking for its archive to be left alone is not asking for an editor's
// lock files to be indexed.
func (o Options) ignored() *ignore.GitIgnore {
	return ignore.CompileIgnoreLines(append(append([]string(nil), DefaultIgnore...), o.Ignore...)...)
}

func (o Options) window() time.Duration {
	if o.Window <= 0 {
		return DefaultWindow
	}
	return o.Window
}

// DefaultExtensions is what counts as a note when nothing says otherwise. It is
// one entry because a default that guesses widely indexes what the user did not
// mean.
var DefaultExtensions = []string{".md"}

// DefaultIgnore is what no vault has to ask to be left out. A name beginning
// with a dot belongs to a tool rather than to the person — an editor's lock, a
// sync client's bookkeeping.
var DefaultIgnore = []string{".*"}

func (o Options) serviceDir() string {
	if o.ServiceDir == "" {
		return DefaultServiceDir
	}
	return o.ServiceDir
}

func (o Options) extensions() []string {
	if len(o.Extensions) == 0 {
		return DefaultExtensions
	}
	return o.Extensions
}

func (o Options) isNote(name string) bool {
	for _, ext := range o.extensions() {
		if len(name) > len(ext) && strings.EqualFold(name[len(name)-len(ext):], ext) {
			return true
		}
	}
	return false
}
