// Package vaultreading runs the three passes that make a vault searchable.
package vaultreading

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// A Task is one unit of work as presented to the user.
type Task struct {
	ID string

	Title string
	About string

	Count, Total int64

	Error string

	IsUserStarted bool

	Kind Kind

	Unit Unit

	PluginID, PluginName string

	Cancel context.CancelFunc
}

// Kind is what an entry of the list of tasks is.
type Kind int

const (
	// KindWork is a piece of work that runs, stops, or fails.
	KindWork Kind = iota
	// KindNotice is a message of the core. It is neither running nor failed.
	KindNotice
)

// Unit is what a piece of work counts. Bytes are read out in the sizes a
// person reads them in, and everything else is counted one by one.
type Unit int

const (
	Things Unit = iota
	Bytes
	// Seconds are a stretch of a recording, read out as a length of time.
	Seconds
)

// ScanResult reports what a scan did, in the terms the user cares about.
type ScanResult struct {
	Notes      int
	Assets     int
	Indexed    int
	Unchanged  int
	Removed    int
	Vanished   int
	Unreadable int
}

// IndexResult reports what indexing did.
type IndexResult struct {
	Seen          int
	Recorded      int
	Unchanged     int
	Extracted     int
	Chunks        int
	Unreadable    int
	Removed       int
	Outdated      int
	Vanished      int
	CurrentSource string
}

// EmbedResult reports what embedding did.
type EmbedResult struct {
	Pending       int
	Embedded      int
	Reused        int
	Vanished      int
	Displaced     int
	Skipped       int
	CurrentSource string
	Active        []string
	IsBusy        bool
}

// NoteScanner scans the notes of a vault.
type NoteScanner interface {
	ScanWithProgress(ctx context.Context, v domain.Vault, onProgress func(ScanResult)) (ScanResult, error)
}

// TextIndexer indexes the file texts of a vault.
type TextIndexer interface {
	IndexWithProgress(
		ctx context.Context, v domain.Vault, onProgress func(IndexResult), isSkipped func(path string) bool,
	) (IndexResult, error)
	IndexOne(ctx context.Context, v domain.Vault, path string) (IndexResult, error)
}

// VectorEmbedder embeds the chunks of a vault.
type VectorEmbedder interface {
	EmbedWithProgress(
		ctx context.Context, v domain.Vault, onProgress func(EmbedResult), isSkipped func(path, hash string) bool,
	) (EmbedResult, error)
}

// ReadWholeVault makes a vault answer: its notes are read, the text versions of
// its other files are cut, and what was cut is given its vectors.
type ReadWholeVault struct {
	notes   NoteScanner
	texts   TextIndexer
	vectors VectorEmbedder

	OnNotesProgress func(ScanResult)

	OnTextsProgress func(IndexResult)

	OnVectorsProgress func(EmbedResult)

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
	Notes   ScanResult
	Texts   IndexResult
	Vectors EmbedResult
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
func (u ReadWholeVault) IndexTexts(ctx context.Context, v domain.Vault) (IndexResult, error) {
	res, err := u.texts.IndexWithProgress(ctx, v, u.OnTextsProgress, u.IsTextSkipped)
	if err != nil {
		return res, fmt.Errorf("indexing the file texts of %s: %w", v.Name, err)
	}
	return res, nil
}

// MakeVectors embeds every chunk that lacks a vector.
func (u ReadWholeVault) MakeVectors(ctx context.Context, v domain.Vault) (EmbedResult, error) {
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

// FindRemovedSources is the sources other than notes the index holds at and
// under the paths, with the content hash each was read with. A source never
// read has no hash and matches nothing.
func FindRemovedSources(
	ctx context.Context, queries sourceport.RemovalLookup, vaultID domain.VaultID, paths []string,
) ([]source.Text, error) {
	var out []source.Text
	for _, path := range paths {
		under, err := queries.GetSourcesUnder(ctx, vaultID, path)
		if err != nil {
			return nil, err
		}
		for _, ref := range under {
			if ref.Kind == domain.KindNote {
				continue
			}
			text, ok, err := queries.GetSourceText(ctx, vaultID, ref.Path)
			if err != nil {
				return nil, err
			}
			if ok && text.ContentHash != "" {
				out = append(out, text)
			}
		}
	}
	return out, nil
}

// MoveRenamedSources moves each removed source to the path among added whose
// file holds the same content, and returns the paths it moved from. Only a
// file of the same size as a removed source is read.
func MoveRenamedSources(
	ctx context.Context,
	sources sourceport.SourceRepository,
	reader vaultport.VaultReader,
	vaultID domain.VaultID,
	removed []source.Text,
	added []string,
) ([]string, error) {
	var moved []string
	for _, path := range added {
		if len(removed) == 0 {
			break
		}
		i := findSameContent(ctx, reader, path, removed)
		if i < 0 {
			continue
		}
		if err := sources.MoveSources(ctx, vaultID, removed[i].Path, path); err != nil {
			return moved, err
		}
		moved = append(moved, removed[i].Path)
		removed = slices.Delete(removed, i, i+1)
	}
	return moved, nil
}

func findSameContent(ctx context.Context, reader vaultport.VaultReader, path string, removed []source.Text) int {
	ref, err := reader.StatFile(ctx, path)
	if err != nil || !slices.ContainsFunc(removed, func(old source.Text) bool { return old.Size == ref.Size }) {
		return -1
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return -1
	}
	hash := domain.HashContent(raw)
	return slices.IndexFunc(removed, func(old source.Text) bool { return old.ContentHash == hash })
}
