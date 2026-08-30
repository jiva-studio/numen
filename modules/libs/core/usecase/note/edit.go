package note

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrNoNote is what an operation gets when the vault holds no note at the path
// it was given. A vault that cannot be reached at all comes back as it arrived.
var ErrNoNote = errors.New("the vault holds no note at this path")

// ErrUnlevelled is a file that was written and an index that could not be
// brought up to date with it. The vault holds what the write put there, and the
// path is levelled again at the next scan.
var ErrUnlevelled = errors.New("the vault was written and the index was not brought level with it")

// Levelled is what bringing the index up to date came to, said so that a caller
// can tell it from a write that never landed.
func Levelled(path string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %w", ErrUnlevelled, path, err)
}

// missing is ErrNoNote where the vault holds no note at the path, and the error
// as it arrived otherwise.
func missing(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNoNote
	}
	return err
}

// editing is what every change to the contents of an existing note needs.
//
// It is one function because the shape is always the same and the rules in it
// are easy to forget one at a time: read, refuse what cannot be read, write the
// identifier because this is an edit, change the one thing, put it back, and
// bring the index level.
type editing struct {
	readers     port.VaultReaders
	writers     port.VaultWriters
	index       func(ctx context.Context, v domain.Vault, paths []string) error
	now         func() time.Time
	fingerprint domain.FileRef
	// overwrite is a caller writing what is in front of the person: the note
	// on disk is replaced without being held to a fingerprint, a note that is
	// not there is made, and no identifier is stamped.
	overwrite bool
	// seen is what the caller last saw of the note, and is what a file that is
	// there is compared with. Nil for a caller that puts its text down whatever
	// the note now holds.
	seen *Seen
}

func (e editing) apply(ctx context.Context, v domain.Vault, path string, change func(*markdown.Document) error) (domain.FileRef, error) {
	written, err := e.splice(ctx, v, path, change)
	if err != nil {
		return domain.FileRef{}, err
	}
	if e.index == nil {
		return written, nil
	}
	return written, e.index(ctx, v, []string{path})
}

// splice is the read, the change and the write, under this vault's write lock
// from before the read until after the file is replaced.
func (e editing) splice(
	ctx context.Context, v domain.Vault, path string, change func(*markdown.Document) error,
) (domain.FileRef, error) {
	release, err := e.writers.Hold(ctx, v)
	if err != nil {
		return domain.FileRef{}, err
	}
	defer release()

	reader, err := e.readers.Open(v)
	if err != nil {
		return domain.FileRef{}, err
	}

	// What the file is, asked before its bytes are read, which is the order a
	// read hands its fingerprint over in.
	var on domain.FileRef
	var looked error
	if e.seen != nil || (e.fingerprint == (domain.FileRef{}) && !e.overwrite) {
		on, looked = reader.Stat(ctx, path)
	}

	raw, err := reader.Read(ctx, path)
	switch {
	case err == nil:
	case e.overwrite && errors.Is(err, fs.ErrNotExist):
		// The note is made by this write, out of the person's own text and
		// nothing else: no frontmatter, and no identifier. An identifier
		// arrives when the application changes a note's contents.
		raw = nil
	default:
		return domain.FileRef{}, fmt.Errorf("read %s: %w", path, missing(err))
	}

	// Every edit is a read, a think and a write, and the person may save the
	// note in their own editor in between. A caller that said what it believed
	// the note was is held to that; one that said nothing is held to what was
	// read just now. A caller writing over what is there is held to neither.
	against := e.fingerprint
	if against == (domain.FileRef{}) && !e.overwrite {
		if looked != nil {
			return domain.FileRef{}, fmt.Errorf("look at %s: %w", path, missing(looked))
		}
		against = on
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		return domain.FileRef{}, fmt.Errorf("%s: %w", path, err)
	}

	// A note that is not there cannot hold anything the caller has not read, so
	// it is made.
	if looked == nil && e.seen.stale(on, markdown.Normalised(doc.Body())) {
		return domain.FileRef{}, fmt.Errorf("write %s: %w", path, port.ErrChanged)
	}

	if err := change(doc); err != nil {
		return domain.FileRef{}, fmt.Errorf("%s: %w", path, err)
	}

	// The application is editing this note, so it may write the identifier the
	// note does not have. Writing one is what editing a note permits and reading
	// past it does not: nothing backfills an identifier into a note it only read.
	// A caller writing over what is there is the person editing their own note,
	// and leaves the frontmatter as they wrote it.
	if _, carried := doc.Identifier(); !carried && !e.overwrite {
		at := time.Now
		if e.now != nil {
			at = e.now
		}
		identifier, err := ulid.New(at())
		if err != nil {
			return domain.FileRef{}, err
		}
		if err := doc.SetIdentifier(identifier); err != nil {
			return domain.FileRef{}, err
		}
	}

	writer, err := e.writers.Open(v)
	if err != nil {
		return domain.FileRef{}, err
	}
	return writer.Write(ctx, path, doc.Bytes(), against)
}
