package port

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
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
// is one that resolves out of it through a link. Where a name is kept is the
// store's own business.
type DerivedStore interface {
	// ReadArtifact returns what is stored under a name. A name with nothing
	// under it gets fs.ErrNotExist: the folder is on the person's disk and they
	// may empty it, which is an answer and not a failure.
	ReadArtifact(ctx context.Context, name domain.ArtifactName) ([]byte, error)

	// OpenArtifact is one file to read a part of, and how many bytes it holds.
	// A copy of a video is played from the middle and is an hour long, and
	// holding all of it to answer for a second of it is the whole file in
	// memory for every listener.
	//
	// The same errors as ReadArtifact. Whoever opens it closes it.
	OpenArtifact(ctx context.Context, name domain.ArtifactName) (io.ReadSeekCloser, int64, error)

	// WriteArtifact puts content under a name, atomically, replacing whatever
	// was there. Replacing is ordinary: one recognition run twice writes the
	// same bytes under the same name.
	WriteArtifact(ctx context.Context, name domain.ArtifactName, content []byte) error

	// WriteArtifactStream puts what a reader gives under a name, atomically. A
	// copy of a video arrives over a network and is not held in memory to be
	// written.
	WriteArtifactStream(ctx context.Context, name domain.ArtifactName, from io.Reader) (int64, error)

	// AppendArtifact adds to what is under a name, creating it when there is
	// nothing. A recognition is written as it is read, over an hour, and
	// reading a growing file back in order to rewrite it costs the square of
	// its pages.
	AppendArtifact(ctx context.Context, name domain.ArtifactName, content []byte) error

	// RemoveArtifact takes a name out of the store. A name already gone is the
	// outcome that was asked for.
	RemoveArtifact(ctx context.Context, name domain.ArtifactName) error

	// ClaimArtifact takes a name for this caller alone and returns what lets it
	// go. A name already claimed is refused with ErrClaimed, so work whose whole
	// output is one file is done once.
	ClaimArtifact(ctx context.Context, name domain.ArtifactName) (release func() error, err error)
}

// DerivedStores opens one vault's store. Which vault a use case works on is
// decided while it runs, as with readers and writers.
type DerivedStores interface {
	Open(v domain.Vault) (DerivedStore, error)
}
