package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// ReadWholeVault makes a vault answer: its notes are read, its books are read,
// and what was cut is given its vectors.
//
// The three are one use case because they are one order, and because a caller
// that ran two of them left a vault that answers by its words alone. Each
// carries its own progress, so what a person is shown is set on the pass it
// belongs to.
type ReadWholeVault struct {
	Notes   Scan
	Books   source.Extract
	Vectors source.Embed
}

// NewReadWholeVault is the three passes, in the order they stand in: the notes
// walked, the books read, and the vectors made.
//
// All three are named here because a vault read short of one of them answers,
// and answers by less than it holds — a search that misses a book, or misses
// every word a person did not type themselves.
func NewReadWholeVault(notes Scan, books source.Extract, vectors source.Embed) ReadWholeVault {
	return ReadWholeVault{Notes: notes, Books: books, Vectors: vectors}
}

// ReadWholeVaultResult is what each pass did. A pass that did not run is a zero
// value, and nothing here tells that apart from a pass that found nothing.
type ReadWholeVaultResult struct {
	Notes   ScanResult
	Books   source.ExtractResult
	Vectors source.EmbedResult
}

// Execute reads the whole vault, in the order the three passes stand in.
//
// The notes come first: a vault is useful the moment its notes answer, and a
// library takes minutes to cut. A library that could not be read is said and
// does not stop the vectors — the notes are already cut and owe theirs, and
// both failures come back together.
func (u ReadWholeVault) Execute(ctx context.Context, v domain.Vault) (ReadWholeVaultResult, error) {
	var out ReadWholeVaultResult

	notes, err := u.Notes.Execute(ctx, v)
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
	res, err := u.Books.Execute(ctx, v)
	if err != nil {
		return res, fmt.Errorf("reading the books of %s: %w", v.Name, err)
	}
	return res, nil
}

// MakeVectors gives every chunk that owes a vector one.
func (u ReadWholeVault) MakeVectors(ctx context.Context, v domain.Vault) (source.EmbedResult, error) {
	res, err := u.Vectors.Execute(ctx, v)
	if err != nil {
		return res, fmt.Errorf("embedding %s: %w", v.Name, err)
	}
	return res, nil
}

// CutOne cuts one source again from whatever its text now says. A recognition
// writes a batch of pages and asks for this, so a book being read answers about
// the pages that have been read.
func (u ReadWholeVault) CutOne(ctx context.Context, v domain.Vault, path string) error {
	if _, err := u.Books.One(ctx, v, path); err != nil {
		return fmt.Errorf("cutting %s: %w", path, err)
	}
	return nil
}
