package editor

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// A source carrying no text a person typed is put through a run: a scan is
// read, a recording is heard. How far one has got is read off the store: what
// a run writes and what it holds while it writes are both names in it.
//
// A source a person named is never turned away for want of a turn. It waits in
// line and is taken up before anything the vault set itself.

// A Runner reads a source a model has to produce the text of. Recognising a scan
// and hearing a recording are one shape here, and one at a time of each.
type Runner interface {
	// Start takes on one source a person named, and says whether it began now
	// or waits its turn. It runs under the application, so whoever asked is
	// answered at once.
	Start(v domain.Vault, path string) source.StartOutcome
}
