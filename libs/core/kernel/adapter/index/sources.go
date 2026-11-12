package index

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/version"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// sources answers the source and vector ports out of the chunk tables.
//
// It translates between the core's types and the SQLite schema: a source
// arrives as a file reference and is stored as columns, and a passage is read
// back as a row and returned as a location in a file.

type sources struct {
	queries
	write    *chunk.Repository
	versions *version.Repository
}

// queries is the read-only half of sources: what the index holds about sources,
// and which chunks need a vector. A caller that receives only queries cannot
// write.
type queries struct {
	files   *chunk.Files
	vectors *chunk.Vectors
}

// SaveSource records the current state of a file with no recipe: a source saved
// this way needs its text extracted.
func (s sources) SaveSource(ctx context.Context, vaultID domain.VaultID, src domain.Source) error {
	return s.write.SaveSource(ctx, vaultID, newChunkSource(src))
}

// SaveExtraction records the source and replaces its chunks in one write, so a
// recipe is never recorded for chunks that are not there.
func (s sources) SaveExtraction(ctx context.Context, vaultID domain.VaultID, e domain.SourceChunks) error {
	return s.write.SaveExtraction(ctx, vaultID, newChunkSource(e.Source), chunk.FromDomain(e.Chunks))
}

// RemoveSources takes out the sources at the paths given, and their chunks and
// vectors with them.
func (s sources) RemoveSources(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind, paths []string) error {
	return s.write.RemoveSources(ctx, vaultID, string(kind), paths)
}

// MoveSources files what was at one path, and everything under it, where it now
// is. A note its filename names is called by the one it lands under. The text
// versions of what moved move with it.
func (s sources) MoveSources(ctx context.Context, vaultID domain.VaultID, from, to string) error {
	if err := s.write.MoveSources(ctx, vaultID, from, to); err != nil {
		return err
	}
	return s.versions.MoveVersions(ctx, vaultID, from, to)
}

// MoveVersions files the text versions of what was at one path, and of everything
// under it, at the path it now has.
func (s sources) MoveVersions(ctx context.Context, vaultID domain.VaultID, from, to string) error {
	return s.versions.MoveVersions(ctx, vaultID, from, to)
}

// GetSourcesUnder is every source the vault holds at a path and beneath it.
func (s queries) GetSourcesUnder(ctx context.Context, vaultID domain.VaultID, path string) ([]domain.FileStat, error) {
	return s.files.GetSourcesUnder(ctx, vaultID, path)
}

func (s queries) GetFileStats(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind) (map[string]domain.FileStat, error) {
	return s.files.GetFileStats(ctx, vaultID, string(kind))
}

func (s queries) GetUnchunkedSources(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind, limit int) ([]string, error) {
	return s.files.GetUnchunkedSources(ctx, vaultID, string(kind), limit)
}

func (s queries) GetSourcesWithOtherRecipe(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind, recipes []string, limit int) ([]string, error) {
	return s.files.GetSourcesWithOtherRecipe(ctx, vaultID, string(kind), recipes, limit)
}

// SaveVectors writes a group: the coarse row of each, and the vector itself
// where the text it was made from addresses it.
func (s sources) SaveVectors(ctx context.Context, vectors []vector.Stored) error {
	out := make([]chunk.Vector, 0, len(vectors))
	for _, v := range vectors {
		row, err := chunk.Row(v.ChunkID)
		if err != nil {
			return err
		}
		out = append(out, chunk.Vector{
			Chunk:  row,
			Hash:   v.Hash,
			Recipe: v.Recipe,
			Value:  v.Value,
			Coarse: v.Coarse,
		})
	}
	return s.write.SaveVectors(ctx, out)
}

func (s queries) GetUnembeddedChunks(ctx context.Context, vaultID domain.VaultID, recipe string, after vector.ChunkCursor, limit int) ([]domain.Passage, vector.ChunkCursor, error) {
	from, err := parseCursor(after)
	if err != nil {
		return nil, "", err
	}
	found, err := s.vectors.GetUnembeddedChunks(ctx, vaultID, recipe, from, limit)
	if err != nil {
		return nil, "", err
	}
	out := make([]domain.Passage, 0, len(found))
	for _, p := range found {
		out = append(out, domain.Passage{
			ChunkID:     chunk.ID(p.Chunk),
			Source:      p.Path,
			Producer:    p.Producer,
			ContentHash: p.ContentHash,
			Start:       p.Start,
			Length:      p.Length,
			Location:    p.Location,
			ChunkHash:   p.ChunkHash,
		})
	}
	next := after
	if len(found) > 0 {
		last := found[len(found)-1]
		next = vector.ChunkCursor(strconv.FormatInt(last.MTime, 10) + ":" + strconv.FormatInt(last.Source, 10) + ":" + strconv.FormatInt(last.Chunk, 10))
	}
	return out, next, nil
}

func parseCursor(after vector.ChunkCursor) (chunk.Cursor, error) {
	if after == "" {
		return chunk.Cursor{MTime: math.MaxInt64, Source: math.MaxInt64}, nil
	}
	parts := strings.Split(string(after), ":")
	if len(parts) == 1 {
		row, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return chunk.Cursor{}, fmt.Errorf("%q is no chunk of this index", after)
		}
		return chunk.Cursor{MTime: math.MaxInt64, Source: math.MaxInt64, Chunk: row}, nil
	}
	numbers, err := parseCursorNumbers(after, parts)
	if err != nil {
		return chunk.Cursor{}, err
	}
	switch len(numbers) {
	case 2:
		return chunk.Cursor{MTime: numbers[0], Source: math.MaxInt64, Chunk: numbers[1]}, nil
	case 3:
		return chunk.Cursor{MTime: numbers[0], Source: numbers[1], Chunk: numbers[2]}, nil
	}
	return chunk.Cursor{}, fmt.Errorf("%q is no cursor of this index", after)
}

func parseCursorNumbers(after vector.ChunkCursor, parts []string) ([]int64, error) {
	if len(parts) != 2 && len(parts) != 3 {
		return nil, fmt.Errorf("%q is no cursor of this index", after)
	}
	numbers := make([]int64, len(parts))
	for i, part := range parts {
		number, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%q is no cursor of this index", after)
		}
		numbers[i] = number
	}
	return numbers, nil
}

func newChunkSource(s domain.Source) chunk.Source {
	return chunk.Source{
		Path:        s.FileStat.Path,
		Kind:        string(s.FileStat.Kind),
		Size:        s.FileStat.Size,
		MTime:       chunk.ConvertTimeToNanos(s.FileStat.ModTime),
		ContentHash: s.ContentHash,
		Recipe:      s.Recipe,
		Producer:    s.Producer,
	}
}

// GetKeptVectors is the vectors already made for these texts under this recipe.
func (s sources) GetKeptVectors(ctx context.Context, recipe string, of [][]byte) (map[string][]byte, error) {
	return s.vectors.GetKeptVectors(ctx, recipe, of)
}

// GetSourceText is what one source's text came from, and false where the index holds
// no source at that path.
func (s queries) GetSourceText(ctx context.Context, vaultID domain.VaultID, path string) (sourcedomain.Text, bool, error) {
	found, target, err := s.files.GetSourceText(ctx, vaultID, path)
	if err != nil || !target {
		return sourcedomain.Text{}, false, err
	}
	return sourcedomain.Text{
		FileStat:    domain.FileStat{Path: found.Path, Size: found.Size, ModTime: chunk.ConvertNanosToTime(found.MTime)},
		Producer:    found.Producer,
		ContentHash: found.ContentHash,
	}, true, nil
}

// GetRecognizedSources is the sources of one kind whose text a producer made.
func (s queries) GetRecognizedSources(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind) ([]sourcedomain.Text, error) {
	found, err := s.files.GetRecognizedSources(ctx, vaultID, string(kind))
	if err != nil {
		return nil, err
	}
	out := make([]sourcedomain.Text, 0, len(found))
	for _, r := range found {
		out = append(out, sourcedomain.Text{
			FileStat:    domain.FileStat{Path: r.Path},
			Producer:    r.Producer,
			ContentHash: r.ContentHash,
		})
	}
	return out, nil
}
