package filesystem

import "strings"

// Options are what a vault on disk may be configured with.
type Options struct {
	// ServiceDir is the folder the application keeps its own files in. Empty
	// means the default.
	ServiceDir string
	// Extensions are the file extensions treated as notes, with the leading
	// dot. Empty means the default, which is markdown alone.
	Extensions []string
}

// DefaultExtensions is what counts as a note when nothing says otherwise. It is
// one entry because a default that guesses widely indexes what the user did not
// mean.
var DefaultExtensions = []string{".md"}

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
