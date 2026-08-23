package index

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// sources answers the source and vector ports out of the chunk tables.
//
// It exists because the core says what it needs in its own words and this
// package says what SQLite holds in its own, and the two are close without being
// the same: a source arrives here as a file reference and is stored as columns,
// and a passage is read back as a row and handed over as a place in a file.

type sources struct {
	write *chunk.Repository
	read  *chunk.Queries
}

// SaveSource records what a file is now, with no recipe: a source saved this way
// owes its text.
func (s sources) SaveSource(ctx context.Context, vaultID string, src port.Source) error {
	return s.write.SaveSource(ctx, vaultID, stored(src))
}

// SaveExtraction records the source and replaces its chunks in one write, so a
// recipe is never recorded for chunks that are not there.
func (s sources) SaveExtraction(ctx context.Context, vaultID string, e port.Extraction) error {
	return s.write.SaveExtraction(ctx, vaultID, stored(e.Source), windows(e.Windows))
}

// RemoveSources takes out the sources at the paths given, and their chunks and
// vectors with them.
func (s sources) RemoveSources(ctx context.Context, vaultID string, kind domain.SourceKind, paths []string) error {
	return s.write.RemoveSources(ctx, vaultID, string(kind), paths)
}

func (s sources) Fingerprints(ctx context.Context, vaultID string, kind domain.SourceKind) (map[string]domain.FileRef, error) {
	return s.read.Fingerprints(ctx, vaultID, string(kind))
}

func (s sources) Unchunked(ctx context.Context, vaultID string, kind domain.SourceKind, limit int) ([]string, error) {
	return s.read.Unchunked(ctx, vaultID, string(kind), limit)
}

func (s sources) ByOtherRecipe(ctx context.Context, vaultID string, kind domain.SourceKind, recipes []string, limit int) ([]string, error) {
	return s.read.ByOtherRecipe(ctx, vaultID, string(kind), recipes, limit)
}

// SaveVectors writes a group: the coarse row of each, and the vector itself
// where the text it was made from addresses it.
func (s sources) SaveVectors(ctx context.Context, vectors []port.Vector) error {
	out := make([]chunk.Vector, 0, len(vectors))
	for _, v := range vectors {
		out = append(out, chunk.Vector{
			Chunk:       v.Chunk,
			Fingerprint: v.Fingerprint,
			Recipe:      v.Model.Recipe(),
			Value:       v.Value,
			Coarse:      v.Coarse,
		})
	}
	return s.write.SaveVectors(ctx, out)
}

func (s sources) Unembedded(ctx context.Context, vaultID string, model port.EmbeddingModel, after int64, limit int) ([]domain.Passage, error) {
	found, err := s.read.Unembedded(ctx, vaultID, model.Recipe(), after, limit)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Passage, 0, len(found))
	for _, p := range found {
		out = append(out, domain.Passage{
			Chunk:       p.Chunk,
			Source:      p.Path,
			TextFrom:    p.TextFrom,
			Hash:        p.Hash,
			Start:       p.Start,
			Length:      p.Length,
			Location:    p.Location,
			Fingerprint: p.Fingerprint,
		})
	}
	return out, nil
}

func stored(s port.Source) chunk.Source {
	return chunk.Source{
		Path:     s.Ref.Path,
		Kind:     string(s.Ref.Kind),
		Size:     s.Ref.Size,
		MTime:    s.Ref.MTime,
		Hash:     s.Hash,
		Recipe:   s.Recipe,
		TextFrom: s.TextFrom,
	}
}

func windows(in []port.Window) []chunk.Window {
	out := make([]chunk.Window, 0, len(in))
	for _, w := range in {
		out = append(out, chunk.Window{
			Start:    w.Start,
			Length:   w.Length,
			Location: w.Location,
			Text:     w.Text,
			Opens:    w.Opens,
			Small:    windows(w.Small),
		})
	}
	return out
}

// Kept is the vectors already made for these texts under this recipe.
func (s sources) Kept(ctx context.Context, recipe string, of [][]byte) (map[string][]byte, error) {
	return s.read.Kept(ctx, recipe, of)
}

// Reading is what one source's text came from, and false where the index holds
// no source at that path.
func (s sources) Reading(ctx context.Context, vaultID, path string) (port.Recognised, bool, error) {
	found, held, err := s.read.Reading(ctx, vaultID, path)
	if err != nil || !held {
		return port.Recognised{}, false, err
	}
	return port.Recognised{
		Path: found.Path, From: found.From, Hash: found.Hash,
		Size: found.Size, MTime: found.MTime,
	}, true, nil
}

// Recognised is the sources of one kind whose text a producer made.
func (s sources) Recognised(ctx context.Context, vaultID string, kind domain.SourceKind) ([]port.Recognised, error) {
	found, err := s.read.Recognised(ctx, vaultID, string(kind))
	if err != nil {
		return nil, err
	}
	out := make([]port.Recognised, 0, len(found))
	for _, r := range found {
		out = append(out, port.Recognised{Path: r.Path, From: r.From, Hash: r.Hash})
	}
	return out, nil
}
