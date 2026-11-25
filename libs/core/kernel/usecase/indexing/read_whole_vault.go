package indexing

import (
	"context"
	"errors"
	"fmt"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
)

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

// EmbedVectors embeds the chunks of a vault.
type EmbedVectors func(
	ctx context.Context, v domain.Vault, onProgress func(vaultreading.EmbedResult), isSkipped func(path, hash string) bool,
) (vaultreading.EmbedResult, error)

// VaultPreparer prepares a ReadWholeVault for a vault once it has been found.
type VaultPreparer = func(ctx context.Context, v domain.Vault) (ReadWholeVault, error)

// ReadWholeVault makes a vault answer: its notes are read, the text versions of
// its other files are cut, and what was cut is given its vectors.
type ReadWholeVault struct {
	lookup  vaultport.VaultLookup
	prepare VaultPreparer
	notes   NoteScanner
	texts   TextIndexer
	vectors EmbedVectors

	Unembedded error
}

// ReadCallbacks is what one read reports to its caller, and the sources it leaves out. A nil function
// reports nothing and leaves out nothing. OnVaultFound receives the vault and the reason it is not embedded.
type ReadCallbacks struct {
	NotifyVaultFound      func(domain.Vault, error)
	NotifyNotesProgress   func(vaultreading.ScanResult)
	NotifyTextsProgress   func(vaultreading.IndexResult)
	NotifyVectorsProgress func(vaultreading.EmbedResult)
	IsTextSkipped         func(path string) bool
	IsVectorSkipped       func(path, hash string) bool
}

// NewReadWholeVault is the three passes, in the order they stand in: the notes
// walked, the file texts cut, and the vectors made.
func NewReadWholeVault(notes NoteScanner, texts TextIndexer, vectors EmbedVectors) ReadWholeVault {
	return ReadWholeVault{notes: notes, texts: texts, vectors: vectors}
}

// NewReadWholeVaultByName creates a ReadWholeVault that resolves the vault and prepares the pass over it.
func NewReadWholeVaultByName(lookup vaultport.VaultLookup, prepare VaultPreparer) ReadWholeVault {
	base := ReadWholeVault{lookup: lookup}
	if prepare == nil {
		prepare = func(context.Context, domain.Vault) (ReadWholeVault, error) { return base, nil }
	}
	base.prepare = prepare
	return base
}

func (u ReadWholeVault) ExecuteByName(
	ctx context.Context, nameOrPath string, reporting ReadCallbacks,
) (domain.Vault, ReadWholeVaultResult, error) {
	v, err := vaultdomain.FindByName(nameOrPath, u.lookup.FindVault)
	if err != nil {
		return domain.Vault{}, ReadWholeVaultResult{}, err
	}
	runner, err := u.prepare(ctx, v)
	if err != nil {
		return domain.Vault{}, ReadWholeVaultResult{}, err
	}
	if runner.Unembedded == nil {
		runner.Unembedded = u.Unembedded
	}
	if reporting.NotifyVaultFound != nil {
		reporting.NotifyVaultFound(v, runner.Unembedded)
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
func (u ReadWholeVault) Execute(ctx context.Context, v domain.Vault, reporting ReadCallbacks) (ReadWholeVaultResult, error) {
	var out ReadWholeVaultResult

	notes, err := u.notes.ScanWithProgress(ctx, v, reporting.NotifyNotesProgress)
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
func (u ReadWholeVault) IndexTexts(ctx context.Context, v domain.Vault, reporting ReadCallbacks) (vaultreading.IndexResult, error) {
	res, err := u.texts.IndexWithProgress(ctx, v, reporting.NotifyTextsProgress, reporting.IsTextSkipped)
	if err != nil {
		return res, fmt.Errorf("indexing the file texts of %s: %w", v.Name, err)
	}
	return res, nil
}

// MakeVectors embeds every chunk that lacks a vector.
func (u ReadWholeVault) MakeVectors(ctx context.Context, v domain.Vault, reporting ReadCallbacks) (vaultreading.EmbedResult, error) {
	res, err := u.vectors(ctx, v, reporting.NotifyVectorsProgress, reporting.IsVectorSkipped)
	if err != nil {
		return res, fmt.Errorf("embedding %s: %w", v.Name, err)
	}
	return res, nil
}

// CutOne re-chunks one file from its current live text version.
func (u ReadWholeVault) CutOne(ctx context.Context, v domain.Vault, path string) (vaultreading.IndexResult, error) {
	res, err := u.texts.IndexOne(ctx, v, path)
	if err != nil {
		return res, fmt.Errorf("cutting %s: %w", path, err)
	}
	return res, nil
}
