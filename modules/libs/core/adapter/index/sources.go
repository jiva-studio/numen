package index

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// sources answers the source and vector ports out of the chunk tables.
//
// It exists because the core says what it needs in its own words and this
// package says what SQLite holds in its own, and the two are close without being
// the same: a source arrives here as a file reference and is stored as columns,
// and a passage is read back as a row and handed over as a place in a file.

type sources struct {
	queries
	write *chunk.Repository
}

// queries is the half of that which is asked: what the index holds about sources,
// and which chunks owe a vector. It is queries alone, so a caller handed one has
// nothing to write with.
type queries struct {
	read *chunk.Queries
}

// SaveSource records what a file is now, with no recipe: a source saved this way
// owes its text.
func (s sources) SaveSource(ctx context.Context, vaultID domain.VaultID, src domain.Source) error {
	return s.write.SaveSource(ctx, vaultID, stored(src))
}

// SaveExtraction records the source and replaces its chunks in one write, so a
// recipe is never recorded for chunks that are not there.
func (s sources) SaveExtraction(ctx context.Context, vaultID domain.VaultID, e domain.SourceChunks) error {
	return s.write.SaveExtraction(ctx, vaultID, stored(e.Source), chunks(e.Chunks))
}

// RemoveSources takes out the sources at the paths given, and their chunks and
// vectors with them.
func (s sources) RemoveSources(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind, paths []string) error {
	return s.write.RemoveSources(ctx, vaultID, string(kind), paths)
}

// MoveSources files what was at one path, and everything under it, where it now
// is. A note its filename names is called by the one it lands under.
func (s sources) MoveSources(ctx context.Context, vaultID domain.VaultID, from, to string) error {
	return s.write.MoveSources(ctx, vaultID, from, to)
}

// Under is every source the vault holds at a path and beneath it.
func (s queries) Under(ctx context.Context, vaultID domain.VaultID, path string) ([]domain.Fingerprint, error) {
	return s.read.Under(ctx, vaultID, path)
}

func (s queries) Fingerprints(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind) (map[string]domain.Fingerprint, error) {
	return s.read.Fingerprints(ctx, vaultID, string(kind))
}

func (s queries) Unchunked(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind, limit int) ([]string, error) {
	return s.read.Unchunked(ctx, vaultID, string(kind), limit)
}

func (s queries) ByOtherRecipe(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind, recipes []string, limit int) ([]string, error) {
	return s.read.ByOtherRecipe(ctx, vaultID, string(kind), recipes, limit)
}

// SaveVectors writes a group: the coarse row of each, and the vector itself
// where the text it was made from addresses it.
func (s sources) SaveVectors(ctx context.Context, vectors []port.Vector) error {
	out := make([]chunk.Vector, 0, len(vectors))
	for _, v := range vectors {
		row, err := chunk.Row(v.ChunkID)
		if err != nil {
			return err
		}
		out = append(out, chunk.Vector{
			Chunk:  row,
			Hash:   v.Hash,
			Recipe: v.Model.Recipe(),
			Value:  v.Value,
			Coarse: v.Coarse,
		})
	}
	return s.write.SaveVectors(ctx, out)
}

func (s queries) Unembedded(ctx context.Context, vaultID domain.VaultID, model port.EmbeddingModel, after port.ChunkCursor, limit int) ([]domain.Passage, port.ChunkCursor, error) {
	from, err := resuming(after)
	if err != nil {
		return nil, "", err
	}
	found, err := s.read.Unembedded(ctx, vaultID, model.Recipe(), from, limit)
	if err != nil {
		return nil, "", err
	}
	out := make([]domain.Passage, 0, len(found))
	for _, p := range found {
		out = append(out, domain.Passage{
			ChunkID:    chunk.ID(p.Chunk),
			Source:     p.Path,
			TextFrom:   p.TextFrom,
			SourceHash: p.Hash,
			Start:      p.Start,
			Length:     p.Length,
			Location:   p.Location,
			ChunkHash:  p.ChunkHash,
		})
	}
	next := after
	if len(found) > 0 {
		next = port.ChunkCursor(chunk.ID(found[len(found)-1].Chunk))
	}
	return out, next, nil
}

// resuming is the chunk a walk carries on after. A cursor is a chunk's own
// address, so it is read the same way, and the empty one is the beginning.
func resuming(after port.ChunkCursor) (int64, error) {
	if after == "" {
		return 0, nil
	}
	return chunk.Row(domain.ChunkID(after))
}

func stored(s domain.Source) chunk.Source {
	return chunk.Source{
		Path:     s.Fingerprint.Path,
		Kind:     string(s.Fingerprint.Kind),
		Size:     s.Fingerprint.Size,
		MTime:    chunk.Stamp(s.Fingerprint.ModTime),
		Hash:     s.Hash,
		Recipe:   s.Recipe,
		TextFrom: s.TextFrom,
	}
}

func chunks(in []domain.Chunk) []chunk.Chunk {
	out := make([]chunk.Chunk, 0, len(in))
	for _, c := range in {
		out = append(out, chunk.Chunk{
			Start:    c.Start,
			Length:   c.Length,
			Location: c.Location,
			Text:     c.Text,
			Opens:    c.Opens,
			Small:    chunks(c.Small),
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
func (s queries) Reading(ctx context.Context, vaultID domain.VaultID, path string) (port.SourceText, bool, error) {
	found, held, err := s.read.Reading(ctx, vaultID, path)
	if err != nil || !held {
		return port.SourceText{}, false, err
	}
	return port.SourceText{
		Fingerprint: domain.Fingerprint{Path: found.Path, Size: found.Size, ModTime: chunk.Instant(found.MTime)},
		Producer:    found.Producer,
		Hash:        found.Hash,
	}, true, nil
}

// Recognised is the sources of one kind whose text a producer made.
func (s queries) Recognised(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind) ([]port.SourceText, error) {
	found, err := s.read.Recognised(ctx, vaultID, string(kind))
	if err != nil {
		return nil, err
	}
	out := make([]port.SourceText, 0, len(found))
	for _, r := range found {
		out = append(out, port.SourceText{
			Fingerprint: domain.Fingerprint{Path: r.Path},
			Producer:    r.Producer,
			Hash:        r.Hash,
		})
	}
	return out, nil
}
