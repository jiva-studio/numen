package indexing

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
)

// NoteScanner scans the notes of a vault.
type NoteScanner interface {
	ScanWithProgress(ctx context.Context, v domain.Vault, onProgress func(vaultreading.ScanResult)) (vaultreading.ScanResult, error)
}

// TextIndexer indexes the file texts of a vault.
type TextIndexer interface {
	IndexWithProgress(
		ctx context.Context, v domain.Vault, onProgress func(vaultreading.IndexResult), isSkipped func(path string) bool,
	) (vaultreading.IndexResult, error)
	IndexOne(ctx context.Context, v domain.Vault, path string) (vaultreading.IndexResult, error)
}

// VectorEmbedder embeds the chunks of a vault.
type VectorEmbedder interface {
	EmbedWithProgress(
		ctx context.Context, v domain.Vault, onProgress func(vaultreading.EmbedResult), isSkipped func(path, hash string) bool,
	) (vaultreading.EmbedResult, error)
}

// VaultFinding resolves what the user typed, a name, a path or an identity, into a vault on the list.
type VaultFinding = func(nameOrPath string) (domain.Vault, error)

// VaultPreparing prepares a ReadWholeVault for a vault once it has been found.
type VaultPreparing = func(ctx context.Context, v domain.Vault) (ReadWholeVault, error)

// ReadWholeVault makes a vault answer: its notes are read, the text versions of
// its other files are cut, and what was cut is given its vectors.
type ReadWholeVault struct {
	find    VaultFinding
	prepare VaultPreparing
	notes   NoteScanner
	texts   TextIndexer
	vectors VectorEmbedder

	Unembedded error
}

// Reporting is what one read reports to its caller, and the sources it leaves out. A nil function
// reports nothing and leaves out nothing. OnVaultFound receives the vault and the reason it is not embedded.
type Reporting struct {
	OnVaultFound      func(domain.Vault, error)
	OnNotesProgress   func(vaultreading.ScanResult)
	OnTextsProgress   func(vaultreading.IndexResult)
	OnVectorsProgress func(vaultreading.EmbedResult)
	IsTextSkipped     func(path string) bool
	IsVectorSkipped   func(path, hash string) bool
}

// NewReadWholeVault is the three passes, in the order they stand in: the notes
// walked, the file texts cut, and the vectors made.
func NewReadWholeVault(notes NoteScanner, texts TextIndexer, vectors VectorEmbedder) ReadWholeVault {
	return ReadWholeVault{notes: notes, texts: texts, vectors: vectors}
}

// NewReadWholeVaultByName creates a ReadWholeVault that resolves the vault and prepares the pass over it.
func NewReadWholeVaultByName(find VaultFinding, prepare VaultPreparing) ReadWholeVault {
	return ReadWholeVault{find: find, prepare: prepare}
}

// ExecuteByName resolves the vault and reads the whole vault.
func (u ReadWholeVault) ExecuteByName(
	ctx context.Context, nameOrPath string, reporting Reporting,
) (domain.Vault, ReadWholeVaultResult, error) {
	v, err := u.find(nameOrPath)
	if err != nil {
		return domain.Vault{}, ReadWholeVaultResult{}, err
	}
	runner := u
	if u.prepare != nil {
		prepared, prepErr := u.prepare(ctx, v)
		if prepErr != nil {
			return domain.Vault{}, ReadWholeVaultResult{}, prepErr
		}
		if prepared.Unembedded == nil {
			prepared.Unembedded = u.Unembedded
		}
		runner = prepared
	}
	if reporting.OnVaultFound != nil {
		reporting.OnVaultFound(v, runner.Unembedded)
	}
	res, err := runner.Execute(ctx, v, reporting)
	return v, res, err
}

// ReadWholeVaultResult is the outcome of each pass. A pass that did not run is
// a zero value, indistinguishable from a pass that found nothing.
type ReadWholeVaultResult struct {
	Notes   vaultreading.ScanResult
	Texts   vaultreading.IndexResult
	Vectors vaultreading.EmbedResult
}

// Execute reads the whole vault, in the order the three passes stand in.
func (u ReadWholeVault) Execute(ctx context.Context, v domain.Vault, reporting Reporting) (ReadWholeVaultResult, error) {
	var out ReadWholeVaultResult

	notes, err := u.notes.ScanWithProgress(ctx, v, reporting.OnNotesProgress)
	out.Notes = notes
	if err != nil {
		return out, fmt.Errorf("reading the notes of %s: %w", v.Name, err)
	}

	texts, read := u.IndexTexts(ctx, v, reporting)
	out.Texts = texts
	// A context that has ended ends the whole pass, and what stopped the texts
	// is what stopped it.
	if read != nil && ctx.Err() != nil {
		return out, read
	}

	vectors, made := u.MakeVectors(ctx, v, reporting)
	out.Vectors = vectors
	return out, errors.Join(read, made)
}

// IndexTexts cuts the live text version of every file of the vault that has one.
func (u ReadWholeVault) IndexTexts(ctx context.Context, v domain.Vault, reporting Reporting) (vaultreading.IndexResult, error) {
	res, err := u.texts.IndexWithProgress(ctx, v, reporting.OnTextsProgress, reporting.IsTextSkipped)
	if err != nil {
		return res, fmt.Errorf("indexing the file texts of %s: %w", v.Name, err)
	}
	return res, nil
}

// MakeVectors embeds every chunk that lacks a vector.
func (u ReadWholeVault) MakeVectors(ctx context.Context, v domain.Vault, reporting Reporting) (vaultreading.EmbedResult, error) {
	res, err := u.vectors.EmbedWithProgress(ctx, v, reporting.OnVectorsProgress, reporting.IsVectorSkipped)
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
