package vaultreading

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// A Task is one piece of work, as a person is shown it.
type Task struct {
	// ID is what this work is called, so that the same work reported again
	// replaces itself.
	ID string

	// Title is the work, in the words to show: "Recognising a scan".
	Title string
	// About is what it is on, and is empty for work that is on nothing in
	// particular.
	About string

	// Count and Total are how far it has got, when there is a total to count
	// against. Both zero is work that is running with nothing to count.
	Count, Total int64

	// Error is why the work stopped, when it stopped badly. A task that stopped
	// badly stays in the list until whoever put it there takes it out.
	Error string

	// IsUserStarted is set for work a person started and is waiting to be told about.
	IsUserStarted bool

	// Unit is what Count and Total are counted in.
	Unit Unit

	// PluginID is the plugin that reported the task, and PluginName its name
	// from the manifest. Both are empty for the core's own work.
	PluginID, PluginName string
}

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
	Notes      int // notes found in the vault
	Assets     int // sources of another kind found in the vault
	Indexed    int // parsed and written, because they were new or had changed
	Unchanged  int // skipped on size and modification time alone
	Removed    int // in the index, no longer on disk
	Vanished   int // walked, but gone by the time it was read
	Unreadable int // walked, present, and the read failed
}

// IndexResult reports what indexing did.
type IndexResult struct {
	Seen          int    // files with text found in the vault
	Recorded      int    // new or changed, so owing their text
	Unchanged     int    // skipped on size and modification time alone
	Extracted     int    // read, chunked and written
	Chunks        int    // chunks written, of both sizes
	Unreadable    int    // read, and no live text version
	Removed       int    // held by the index, no longer in the vault
	Outdated      int    // cut from a version that is no longer the live one
	Vanished      int    // named by the index, vanished by the time it was read
	CurrentSource string // the source open now
}

// EmbedResult reports what embedding did.
type EmbedResult struct {
	Owing         int    // chunks found with no vector from the model in use
	Embedded      int    // chunks that now carry one
	Reused        int    // chunks whose cached vector was claimed
	Vanished      int    // whose source the vault no longer holds
	Displaced     int    // whose place is not in the text their source holds now
	CurrentSource string // the source open now
	IsBusy        bool   // somebody else is embedding this vault, and nothing was done
}

// NoteScanner scans the notes of a vault.
type NoteScanner interface {
	ScanWithProgress(ctx context.Context, v domain.Vault, onProgress func(ScanResult)) (ScanResult, error)
}

// TextIndexer indexes the file texts of a vault.
type TextIndexer interface {
	IndexWithProgress(ctx context.Context, v domain.Vault, onProgress func(IndexResult)) (IndexResult, error)
	IndexOne(ctx context.Context, v domain.Vault, path string) (IndexResult, error)
}

// VectorEmbedder embeds the chunks of a vault.
type VectorEmbedder interface {
	EmbedWithProgress(ctx context.Context, v domain.Vault, onProgress func(EmbedResult)) (EmbedResult, error)
}

// ReadWholeVault makes a vault answer: its notes are read, the text versions of
// its other files are cut, and what was cut is given its vectors.
type ReadWholeVault struct {
	notes   NoteScanner
	texts   TextIndexer
	vectors VectorEmbedder

	// OnNotesProgress, if set, is called as notes are scanned.
	OnNotesProgress func(ScanResult)

	// OnTextsProgress, if set, is called as each file is opened and as each one
	// is written.
	OnTextsProgress func(IndexResult)

	// OnVectorsProgress, if set, is called as each source is embedded.
	OnVectorsProgress func(EmbedResult)
}

// NewReadWholeVault is the three passes, in the order they stand in: the notes
// walked, the file texts cut, and the vectors made.
func NewReadWholeVault(notes NoteScanner, texts TextIndexer, vectors VectorEmbedder) ReadWholeVault {
	return ReadWholeVault{notes: notes, texts: texts, vectors: vectors}
}

// ReadWholeVaultResult is what each pass did. A pass that did not run is a zero
// value, and nothing here tells that apart from a pass that found nothing.
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
	res, err := u.texts.IndexWithProgress(ctx, v, u.OnTextsProgress)
	if err != nil {
		return res, fmt.Errorf("indexing the file texts of %s: %w", v.Name, err)
	}
	return res, nil
}

// MakeVectors gives every chunk that owes a vector one.
func (u ReadWholeVault) MakeVectors(ctx context.Context, v domain.Vault) (EmbedResult, error) {
	res, err := u.vectors.EmbedWithProgress(ctx, v, u.OnVectorsProgress)
	if err != nil {
		return res, fmt.Errorf("embedding %s: %w", v.Name, err)
	}
	return res, nil
}

// CutOne cuts one file again from whatever its live text version now says.
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
	ctx context.Context, queries sourceport.SourceQueries, vaultID domain.VaultID, paths []string,
) ([]sourceport.SourceText, error) {
	var out []sourceport.SourceText
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
			if ok && text.Hash != "" {
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
	removed []sourceport.SourceText,
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

// findSameContent is the index of the removed source whose content the file at
// path holds, and -1 where none does or the file cannot be read.
func findSameContent(ctx context.Context, reader vaultport.VaultReader, path string, removed []sourceport.SourceText) int {
	ref, err := reader.StatFile(ctx, path)
	if err != nil || !slices.ContainsFunc(removed, func(old sourceport.SourceText) bool { return old.Size == ref.Size }) {
		return -1
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return -1
	}
	hash := domain.HashContent(raw)
	return slices.IndexFunc(removed, func(old sourceport.SourceText) bool { return old.Hash == hash })
}
