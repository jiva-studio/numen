package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/usecase/source"
	"github.com/jiva-studio/numen/libs/core/usecase/vectors"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ReadWholeVault makes a vault answer: its notes are read, its books are read,
// and what was cut is given its vectors.
//
// The three are one use case because they are one order, and because a caller
// that ran two of them left a vault that answers by its words alone. Each
// carries its own progress, so what a person is shown is set on the pass it
// belongs to.
type ReadWholeVault struct {
	notes   Scan
	books   source.Extract
	vectors vectors.Embed

	// OnNotesProgress, if set, is called as notes are scanned.
	OnNotesProgress func(ScanResult)

	// OnBooksProgress, if set, is called as each book is opened and as each one is
	// written.
	OnBooksProgress func(source.ExtractResult)

	// OnVectorsProgress, if set, is called as each source is embedded.
	OnVectorsProgress func(vectors.EmbedResult)
}

// NewReadWholeVault is the three passes, in the order they stand in: the notes
// walked, the books read, and the vectors made.
func NewReadWholeVault(notes Scan, books source.Extract, vectors vectors.Embed) ReadWholeVault {
	return ReadWholeVault{notes: notes, books: books, vectors: vectors}
}

// ReadWholeVaultResult is what each pass did. A pass that did not run is a zero
// value, and nothing here tells that apart from a pass that found nothing.
type ReadWholeVaultResult struct {
	Notes   ScanResult
	Books   source.ExtractResult
	Vectors vectors.EmbedResult
}

// Execute reads the whole vault, in the order the three passes stand in.
//
// The notes come first: a vault is useful the moment its notes answer, and a
// library takes minutes to cut. A library that could not be read is said and
// does not stop the vectors — the notes are already cut and owe theirs, and
// both failures come back together.
func (u ReadWholeVault) Execute(ctx context.Context, v domain.Vault) (ReadWholeVaultResult, error) {
	var out ReadWholeVaultResult

	if u.OnNotesProgress != nil {
		u.notes.OnProgress = u.OnNotesProgress
	}
	notes, err := u.notes.Execute(ctx, v)
	out.Notes = notes
	if err != nil {
		return out, fmt.Errorf("reading the notes of %s: %w", v.Name, err)
	}

	books, read := u.ReadBooks(ctx, v)
	out.Books = books
	// A context that has ended ends the whole pass, and what stopped the books
	// is what stopped it.
	if read != nil && ctx.Err() != nil {
		return out, read
	}

	vectors, made := u.MakeVectors(ctx, v)
	out.Vectors = vectors
	return out, errors.Join(read, made)
}

// ReadBooks takes the text out of every book the vault holds and cuts it.
func (u ReadWholeVault) ReadBooks(ctx context.Context, v domain.Vault) (source.ExtractResult, error) {
	if u.OnBooksProgress != nil {
		u.books.OnProgress = u.OnBooksProgress
	}
	res, err := u.books.Execute(ctx, v)
	if err != nil {
		return res, fmt.Errorf("reading the books of %s: %w", v.Name, err)
	}
	return res, nil
}

// MakeVectors gives every chunk that owes a vector one.
func (u ReadWholeVault) MakeVectors(ctx context.Context, v domain.Vault) (vectors.EmbedResult, error) {
	if u.OnVectorsProgress != nil {
		u.vectors.OnProgress = u.OnVectorsProgress
	}
	res, err := u.vectors.Execute(ctx, v)
	if err != nil {
		return res, fmt.Errorf("embedding %s: %w", v.Name, err)
	}
	return res, nil
}

// CutOne cuts one source again from whatever its text now says. A recognition
// writes a batch of pages and asks for this, so a book being read answers about
// the pages that have been read. A note is cut from its own words by the scan,
// so a note is left as it is.
func (u ReadWholeVault) CutOne(ctx context.Context, v domain.Vault, path string) error {
	isNote, err := u.isNote(ctx, v, path)
	if err != nil {
		return err
	}
	if isNote {
		return nil
	}
	return source.NewCutSource(u.books).Execute(ctx, v, path)
}

// isNote reports whether the vault holds a note at the path.
func (u ReadWholeVault) isNote(ctx context.Context, v domain.Vault, path string) (bool, error) {
	if u.notes.readers == nil {
		return false, nil
	}
	reader, err := u.notes.readers.Open(v)
	if err != nil {
		return false, err
	}
	// A path that cannot be stated is left to the cut, which says what stands there.
	ref, statErr := reader.GetFingerprint(ctx, path)
	return statErr == nil && ref.Kind == domain.KindNote, nil
}
