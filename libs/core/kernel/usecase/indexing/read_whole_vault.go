package indexing

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	indexdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/index"
)

// ReadWholeVault makes a vault answer: its notes are read, the text versions of
// its other files are cut, and what was cut is given its vectors.
type ReadWholeVault struct {
	notes   NoteScanner
	texts   TextIndexer
	vectors VectorEmbedder

	OnNotesProgress func(indexdomain.ScanResult)

	OnTextsProgress func(indexdomain.IndexResult)

	OnVectorsProgress func(indexdomain.EmbedResult)

	IsTextSkipped   func(path string) bool
	IsVectorSkipped func(path, hash string) bool
}

// NewReadWholeVault is the three passes, in the order they stand in: the notes
// walked, the file texts cut, and the vectors made.
func NewReadWholeVault(notes NoteScanner, texts TextIndexer, vectors VectorEmbedder) ReadWholeVault {
	return ReadWholeVault{notes: notes, texts: texts, vectors: vectors}
}

// ReadWholeVaultResult is the outcome of each pass. A pass that did not run is
// a zero value, indistinguishable from a pass that found nothing.
type ReadWholeVaultResult struct {
	Notes   indexdomain.ScanResult
	Texts   indexdomain.IndexResult
	Vectors indexdomain.EmbedResult
}

// Execute reads the whole vault, in the order the three passes stand in.
func (u ReadWholeVault) Execute(ctx context.Context, v domain.Vault) (ReadWholeVaultResult, error) {
	var out ReadWholeVaultResult

	notes, err := u.notes.ScanWithProgress(ctx, v, u.OnNotesProgress)
	out.Notes = notes
	if err != nil {
		return out, fmt.Errorf("reading the notes of %s: %w", v.Name, err)
	}

	texts, read := u.IndexTexts(ctx, v)
	out.Texts = texts
	// A context that has ended ends the whole pass, and what stopped the texts
	// is what stopped it.
	if read != nil && ctx.Err() != nil {
		return out, read
	}

	vectors, made := u.MakeVectors(ctx, v)
	out.Vectors = vectors
	return out, errors.Join(read, made)
}

// IndexTexts cuts the live text version of every file of the vault that has one.
func (u ReadWholeVault) IndexTexts(ctx context.Context, v domain.Vault) (indexdomain.IndexResult, error) {
	res, err := u.texts.IndexWithProgress(ctx, v, u.OnTextsProgress, u.IsTextSkipped)
	if err != nil {
		return res, fmt.Errorf("indexing the file texts of %s: %w", v.Name, err)
	}
	return res, nil
}

// MakeVectors embeds every chunk that lacks a vector.
func (u ReadWholeVault) MakeVectors(ctx context.Context, v domain.Vault) (indexdomain.EmbedResult, error) {
	res, err := u.vectors.EmbedWithProgress(ctx, v, u.OnVectorsProgress, u.IsVectorSkipped)
	if err != nil {
		return res, fmt.Errorf("embedding %s: %w", v.Name, err)
	}
	return res, nil
}

// CutOne re-chunks one file from its current live text version.
func (u ReadWholeVault) CutOne(ctx context.Context, v domain.Vault, path string) error {
	if _, err := u.texts.IndexOne(ctx, v, path); err != nil {
		return fmt.Errorf("cutting %s: %w", path, err)
	}
	return nil
}

// NoteScanner scans the notes of a vault.
type NoteScanner interface {
	ScanWithProgress(ctx context.Context, v domain.Vault, onProgress func(indexdomain.ScanResult)) (indexdomain.ScanResult, error)
}

// TextIndexer indexes the file texts of a vault.
type TextIndexer interface {
	IndexWithProgress(
		ctx context.Context, v domain.Vault, onProgress func(indexdomain.IndexResult), isSkipped func(path string) bool,
	) (indexdomain.IndexResult, error)
	IndexOne(ctx context.Context, v domain.Vault, path string) (indexdomain.IndexResult, error)
}

// VectorEmbedder embeds the chunks of a vault.
type VectorEmbedder interface {
	EmbedWithProgress(
		ctx context.Context, v domain.Vault, onProgress func(indexdomain.EmbedResult), isSkipped func(path, hash string) bool,
	) (indexdomain.EmbedResult, error)
}
