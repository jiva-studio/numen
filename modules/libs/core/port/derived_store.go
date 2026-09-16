package port

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// ErrClaimed is what a name another caller holds gets. Work whose whole output
// is one file reads it as that work already being under way.
var ErrClaimed = errors.New("the name is claimed by another caller")

// ErrHeldByAnother is what a name gets when a program outside this one has the
// file open and the system will not open it twice. A synchroniser and a backup
// reader each hold a file for moments at a time, so it is a state to come back
// to and not a failure of the store.
var ErrHeldByAnother = errors.New("the file is held by another program")

// A derived file is one the application made and cannot make again: a model
// read a scan and wrote down what it saw. It is not a note and never becomes
// one — nothing walks it, nothing indexes it as a source of its own, no link
// reaches it, and the person did not write it.
//
// It is kept inside the vault folder because losing it loses work no machine
// here can redo, and in the application's own folder inside it because it is
// the application's and not the person's.
//
// Every name is a name in one store. A name that leaves it is refused, and so
// is one that resolves out of it through a link.
type DerivedStore interface {
	// Read returns what is stored under a name. A name with nothing under it
	// gets fs.ErrNotExist: the folder is on the person's disk and they may
	// empty it, which is an answer and not a failure.
	Read(ctx context.Context, name string) ([]byte, error)

	// Open is one file to read a part of, and how many bytes it holds. A copy of
	// a video is played from the middle and is an hour long, and holding all of
	// it to answer for a second of it is the whole file in memory for every
	// listener.
	//
	// The same errors as Read. Whoever opens it closes it.
	Open(ctx context.Context, name string) (io.ReadSeekCloser, int64, error)

	// Write puts content under a name, atomically, replacing whatever was
	// there. Replacing is ordinary: one recognition run twice writes the same
	// bytes under the same name.
	Write(ctx context.Context, name string, content []byte) error

	// Take puts what a reader gives under a name, atomically. A copy of a video
	// arrives over a network and is not held in memory to be written.
	Take(ctx context.Context, name string, from io.Reader) (int64, error)

	// Append adds to what is under a name, creating it when there is nothing.
	// A reading is written as it is read, over an hour, and reading a
	// growing file back in order to rewrite it costs the square of its pages.
	Append(ctx context.Context, name string, content []byte) error

	// List reports the names the store holds under one of its own, sorted.
	// Work whose whole record is a folder of files nobody names in advance —
	// a run of review writing one file and never touching it again — has no
	// other way to find what is there. A name with nothing under it lists
	// nothing, which is what an empty store answers.
	//
	// The length comes with the name because a file appended to keeps its name
	// and grows: what tells a reader that a file it has already read has
	// changed is how long it now is, and asking that of a listing costs the
	// listing and not the reading.
	List(ctx context.Context, name string) ([]Entry, error)

	// Remove takes a name out of the store. A name already gone is the outcome
	// that was asked for.
	Remove(ctx context.Context, name string) error

	// Claim takes a name for this caller alone and returns what lets it go. A
	// name already claimed is refused with ErrClaimed, so work whose whole
	// output is one file is done once.
	Claim(ctx context.Context, name string) (release func() error, err error)
}

// Entry is one thing a store holds: its name in that store, and how many
// bytes are under it.
type Entry struct {
	Name string
	Size int
}

// DerivedStores opens one vault's store. Which vault a use case works on is
// decided while it runs, as with readers and writers.
type DerivedStores interface {
	Open(v domain.Vault) (DerivedStore, error)
}
