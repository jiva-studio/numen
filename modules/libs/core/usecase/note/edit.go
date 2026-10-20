package note

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
	"github.com/jiva-studio/numen/modules/libs/notes/ulid"
)

// ErrNoNote is what an operation gets when the vault holds no note at the path
// it was given. A vault that cannot be reached at all comes back as it arrived.
var ErrNoNote = errors.New("the vault holds no note at this path")

// ErrUnlevelled is a file that was written and an index that could not be
// brought up to date with it. The vault holds what the write put there, and the
// path is levelled again at the next scan.
var ErrUnlevelled = errors.New("the vault was written and the index was not brought level with it")

// WrapUnlevelled is what bringing the index up to date came to, said so that a
// caller can tell it from a write that never landed. The paths are the ones the
// index was asked about.
func WrapUnlevelled(err error, paths ...string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %w", ErrUnlevelled, strings.Join(paths, ", "), err)
}

// mapMissingNote is ErrNoNote where the vault holds no note at the path, and
// the error as it arrived otherwise.
func mapMissingNote(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNoNote
	}
	return err
}

// Edit is what every change to the contents of an existing note needs.
//
// It is one function because the shape is always the same and the rules in it
// are easy to forget one at a time: read, refuse what cannot be read, write the
// identifier because this is an edit, change the one thing, put it back, and
// bring the index level.
type Edit struct {
	readers port.VaultReaders
	writers port.VaultWriters
	index   Levels
	now     port.Clock

	Fingerprint domain.Fingerprint
	// ShouldOverwrite is a caller writing what is in front of the person: the note
	// on disk is replaced without being held to a fingerprint, a note that is
	// not there is made, and no identifier is stamped.
	ShouldOverwrite bool
	// Bound is the most the file may be. Zero holds it to nothing.
	Bound int
	// Baseline is what the caller last saw of the note, and is what a file that is
	// there is compared with. Nil for a caller that puts its text down whatever
	// the note now holds.
	Baseline *Baseline
}

// NewEdit is what a change to an existing note is made through: the vault it
// is read and written through, what brings it level in the index, and what time
// it is, because an edit stamps the identifier a note arrived without. What the
// caller believes is on disk, and how much of a file it will hold, are set
// beside it.
func NewEdit(
	readers port.VaultReaders, writers port.VaultWriters, index Levels, now port.Clock,
) Edit {
	return Edit{readers: readers, writers: writers, index: index, now: now}
}

// Apply makes one change to the note at path and puts it back.
func (e Edit) Apply(ctx context.Context, v domain.Vault, path string, change func(*markdown.Document) error) (domain.Fingerprint, error) {
	written, err := e.splice(ctx, v, path, change)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	// The file is on disk, so the fingerprint stands beside whatever the
	// levelling came to and a caller can tell the two apart.
	if e.index == nil {
		return written, nil
	}
	return written, WrapUnlevelled(e.index(ctx, v, []string{path}), path)
}

// splice is the read, the change and the write, under this vault's write lock
// from before the read until after the file is replaced.
func (e Edit) splice(
	ctx context.Context, v domain.Vault, path string, change func(*markdown.Document) error,
) (domain.Fingerprint, error) {
	release, err := e.writers.Lock(ctx, v)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	defer release()

	reader, err := e.readers.Open(v)
	if err != nil {
		return domain.Fingerprint{}, err
	}

	// What the file is, asked before its bytes are read, which is the order a
	// read hands its fingerprint over in.
	on, looked := e.statNote(ctx, reader, path)
	raw, err := e.readNote(ctx, reader, path)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	against, err := e.getExpected(path, on, looked)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		return domain.Fingerprint{}, fmt.Errorf("%s: %w", path, err)
	}

	// A note that is not there cannot hold anything the caller has not read, so
	// it is made.
	if looked == nil && e.Baseline.isStale(on, markdown.Normalise(doc.Body())) {
		return domain.Fingerprint{}, fmt.Errorf("write %s: %w", path, port.ErrStale)
	}

	if err := change(doc); err != nil {
		return domain.Fingerprint{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := e.addIdentifier(doc); err != nil {
		return domain.Fingerprint{}, err
	}

	// Every change to a note's contents comes through here, so the size it is
	// written within is asked once, of the file the change came to.
	content := doc.Bytes()
	if err := CheckSize(path, len(content), e.Bound); err != nil {
		return domain.Fingerprint{}, err
	}

	writer, err := e.writers.Open(v)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	return writer.Write(ctx, path, content, against)
}

// statNote is the fingerprint of the file on disk, and the error of asking, for
// a caller that holds the note to what it last saw or to what is there now.
func (e Edit) statNote(ctx context.Context, reader port.VaultReader, path string) (domain.Fingerprint, error) {
	if e.Baseline != nil || (e.Fingerprint.IsZero() && !e.ShouldOverwrite) {
		return reader.GetFingerprint(ctx, path)
	}
	return domain.Fingerprint{}, nil
}

// readNote is the bytes of the note at path. A caller overwriting a note that
// is not there gets no bytes: the write makes the note out of the person's own
// text, with no frontmatter and no identifier.
func (e Edit) readNote(ctx context.Context, reader port.VaultReader, path string) ([]byte, error) {
	raw, err := reader.Read(ctx, path)
	switch {
	case err == nil:
		return raw, nil
	case e.ShouldOverwrite && errors.Is(err, fs.ErrNotExist):
		return nil, nil
	default:
		return nil, fmt.Errorf("read %s: %w", path, mapMissingNote(err))
	}
}

// getExpected is the fingerprint the write is held to: the one the caller gave,
// else the one read just now. A caller writing over what is there is held to
// neither.
func (e Edit) getExpected(path string, on domain.Fingerprint, looked error) (domain.Fingerprint, error) {
	if !e.Fingerprint.IsZero() || e.ShouldOverwrite {
		return e.Fingerprint, nil
	}
	if looked != nil {
		return domain.Fingerprint{}, fmt.Errorf("look at %s: %w", path, mapMissingNote(looked))
	}
	return on, nil
}

// addIdentifier writes the identifier a note arrived without. A caller
// writing over what is there leaves the frontmatter as the person wrote it.
func (e Edit) addIdentifier(doc *markdown.Document) error {
	if _, carried := doc.Identifier(); carried || e.ShouldOverwrite {
		return nil
	}
	return doc.SetIdentifier(ulid.New(e.now()))
}
