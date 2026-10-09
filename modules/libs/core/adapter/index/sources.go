package index

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

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
	return s.write.SaveSource(ctx, vaultID, newChunkSource(src))
}

// SaveExtraction records the source and replaces its chunks in one write, so a
// recipe is never recorded for chunks that are not there.
func (s sources) SaveExtraction(ctx context.Context, vaultID domain.VaultID, e domain.SourceChunks) error {
	return s.write.SaveExtraction(ctx, vaultID, newChunkSource(e.Source), chunks(e.Chunks))
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

// GetSourcesUnder is every source the vault holds at a path and beneath it.
func (s queries) GetSourcesUnder(ctx context.Context, vaultID domain.VaultID, path string) ([]domain.Fingerprint, error) {
	return s.read.GetSourcesUnder(ctx, vaultID, path)
}

func (s queries) Fingerprints(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind) (map[string]domain.Fingerprint, error) {
	return s.read.Fingerprints(ctx, vaultID, string(kind))
}

func (s queries) GetUnchunkedSources(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind, limit int) ([]string, error) {
	return s.read.GetUnchunkedSources(ctx, vaultID, string(kind), limit)
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

func (s queries) GetUnembeddedChunks(ctx context.Context, vaultID domain.VaultID, model port.EmbeddingModel, after port.ChunkCursor, limit int) ([]domain.Passage, port.ChunkCursor, error) {
	fromMTime, fromSource, fromChunk, err := parseCursor(after)
	if err != nil {
		return nil, "", err
	}
	found, err := s.read.GetUnembeddedChunks(ctx, vaultID, model.Recipe(), fromMTime, fromSource, fromChunk, limit)
	if err != nil {
		return nil, "", err
	}
	out := make([]domain.Passage, 0, len(found))
	for _, p := range found {
		out = append(out, domain.Passage{
			ChunkID:    chunk.ID(p.Chunk),
			Source:     p.Path,
			Producer:   p.Producer,
			SourceHash: p.Hash,
			Start:      p.Start,
			Length:     p.Length,
			Location:   p.Location,
			ChunkHash:  p.ChunkHash,
		})
	}
	next := after
	if len(found) > 0 {
		last := found[len(found)-1]
		next = port.ChunkCursor(strconv.FormatInt(last.MTime, 10) + ":" + strconv.FormatInt(last.Source, 10) + ":" + strconv.FormatInt(last.Chunk, 10))
	}
	return out, next, nil
}

// parseCursor is the point in the ordering a walk carries on after. The empty
// cursor starts from the beginning.
func parseCursor(after port.ChunkCursor) (int64, int64, int64, error) {
	if after == "" {
		return math.MaxInt64, math.MaxInt64, 0, nil
	}
	parts := strings.Split(string(after), ":")
	if len(parts) == 1 {
		row, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("%q is no chunk of this index", after)
		}
		return math.MaxInt64, math.MaxInt64, row, nil
	}
	if len(parts) == 2 {
		mtime, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("%q is no cursor of this index", after)
		}
		chunkRow, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("%q is no cursor of this index", after)
		}
		return mtime, math.MaxInt64, chunkRow, nil
	}
	if len(parts) == 3 {
		mtime, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("%q is no cursor of this index", after)
		}
		sourceRow, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("%q is no cursor of this index", after)
		}
		chunkRow, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("%q is no cursor of this index", after)
		}
		return mtime, sourceRow, chunkRow, nil
	}
	return 0, 0, 0, fmt.Errorf("%q is no cursor of this index", after)
}

func newChunkSource(s domain.Source) chunk.Source {
	return chunk.Source{
		Path:     s.Fingerprint.Path,
		Kind:     string(s.Fingerprint.Kind),
		Size:     s.Fingerprint.Size,
		MTime:    chunk.Stamp(s.Fingerprint.ModTime),
		Hash:     s.Hash,
		Recipe:   s.Recipe,
		Producer: s.Producer,
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

// GetKeptVectors is the vectors already made for these texts under this recipe.
func (s sources) GetKeptVectors(ctx context.Context, recipe string, of [][]byte) (map[string][]byte, error) {
	return s.read.GetKeptVectors(ctx, recipe, of)
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

// GetRecognisedSources is the sources of one kind whose text a producer made.
func (s queries) GetRecognisedSources(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind) ([]port.SourceText, error) {
	found, err := s.read.GetRecognisedSources(ctx, vaultID, string(kind))
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
