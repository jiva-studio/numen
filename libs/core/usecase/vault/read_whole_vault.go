package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/usecase/vectors"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ReadWholeVault makes a vault answer: its notes are read, the text versions of
// its other files are cut, and what was cut is given its vectors.
//
// The three are one use case because they are one order, and because a caller
// that ran two of them left a vault that answers by its words alone. Each
// carries its own progress, so what a person is shown is set on the pass it
// belongs to.
type ReadWholeVault struct {
	notes   Scan
	texts   filetext.Index
	vectors vectors.Embed

	// OnNotesProgress, if set, is called as notes are scanned.
	OnNotesProgress func(ScanResult)

	// OnTextsProgress, if set, is called as each file is opened and as each one
	// is written.
	OnTextsProgress func(filetext.IndexResult)

	// OnVectorsProgress, if set, is called as each source is embedded.
	OnVectorsProgress func(vectors.EmbedResult)
}

// NewReadWholeVault is the three passes, in the order they stand in: the notes
// walked, the file texts cut, and the vectors made.
func NewReadWholeVault(notes Scan, texts filetext.Index, vectors vectors.Embed) ReadWholeVault {
	return ReadWholeVault{notes: notes, texts: texts, vectors: vectors}
}

// ReadWholeVaultResult is what each pass did. A pass that did not run is a zero
// value, and nothing here tells that apart from a pass that found nothing.
type ReadWholeVaultResult struct {
	Notes   ScanResult
	Texts   filetext.IndexResult
	Vectors vectors.EmbedResult
}

// Execute reads the whole vault, in the order the three passes stand in.
//
// The notes come first: a vault is useful the moment its notes answer, and a
// library takes minutes to cut. File texts that could not be cut are said and
// do not stop the vectors — the notes are already cut and owe theirs, and
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
func (u ReadWholeVault) IndexTexts(ctx context.Context, v domain.Vault) (filetext.IndexResult, error) {
	if u.OnTextsProgress != nil {
		u.texts.OnProgress = u.OnTextsProgress
	}
	res, err := u.texts.Execute(ctx, v)
	if err != nil {
		return res, fmt.Errorf("indexing the file texts of %s: %w", v.Name, err)
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

// CutOne cuts one file again from whatever its live text version now says. A
// plugin that writes text in batches asks for this, so a file being read
// answers about what has been read. A note with no live text version is left
// with the chunks the scan cut from its own words.
func (u ReadWholeVault) CutOne(ctx context.Context, v domain.Vault, path string) error {
	if _, err := u.texts.IndexOne(ctx, v, path); err != nil {
		return fmt.Errorf("cutting %s: %w", path, err)
	}
	return nil
}
