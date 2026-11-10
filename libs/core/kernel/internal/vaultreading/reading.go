// Package vaultreading runs the three passes that make a vault searchable.
package vaultreading

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
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
