package port

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// ErrNotANote is what a path gets when there is a file at it and the vault does
// not hold that file as a note: an attachment, an export, a folder some tool
// keeps its state in, whatever the vault's own rules leave alone.
//
// It is its own answer and not fs.ErrNotExist. A file that is there and a path
// with nothing at it are different facts, and a caller acts on them
// differently.
var ErrNotANote = errors.New("not a note this vault holds")

// ErrOutside is what a path that does not stay in the vault gets. It is not
// "no note": a caller asking for something outside has made a mistake worth
// hearing about, and "no such note" would hide it.
var ErrOutside = errors.New("not a path inside the vault")

// NoNote reports whether an error says there is no note at the path: nothing is
// there, or something is there that the vault leaves alone.
//
// A caller with work to do on a note asks this. Which of the two it is matters
// to a caller acting on the difference, and it is the difference the two
// sentinels carry.
func NoNote(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, ErrNotANote)
}

// VaultReader is one vault as seen from outside: something that can be walked
// and read. The filesystem is one implementation; a fixture in memory is
// another.
type VaultReader interface {
	// Walk reports every source the vault holds, in unspecified order, each
	// saying which kind it is. The service folder is not reported, and neither
	// is a file of no kind the application reads.
	Walk(ctx context.Context, fn func(domain.FileRef) error) error
	// List reports the entries of one folder, without descending: the files of
	// every kind and the folders under it. The empty path is the vault root.
	//
	// What a walk leaves out is left out here as well. Folders come first,
	// then files, each by name compared without regard to case, and that is
	// the order to draw them in.
	//
	// ErrOutside for a path that leaves the vault, fs.ErrNotExist for a folder
	// that is not there.
	List(ctx context.Context, folder string) ([]domain.Entry, error)
	// Read returns the bytes of one file, addressed by a path a walk reported.
	Read(ctx context.Context, path string) ([]byte, error)
	// Stat answers what a walk reports about one path: its kind, its size and
	// when it changed. It reads no bytes, so it is what a caller asks about a
	// file it does not want to open.
	//
	// fs.ErrNotExist when there is nothing at the path. ErrNotANote when a
	// file is there that the vault leaves alone.
	Stat(ctx context.Context, path string) (domain.FileRef, error)
}

// VaultReaders opens a reader for a given vault. Which vault a use case works
// on is decided while it runs.
type VaultReaders interface {
	Open(v domain.Vault) (VaultReader, error)
}
