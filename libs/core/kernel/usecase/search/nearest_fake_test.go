package search_test

import (
	"context"
	"math"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// nearestFake is an index that answers a search by meaning with the chunks and vectors it was
// given and records what it was asked.
type nearestFake struct {
	mu         sync.Mutex
	candidates []domain.ChunkID
	vectors    map[domain.ChunkID][]byte
	coarseErr  error
	storedErr  error
	coarse     []source.CoarseQuery
	stored     []source.StoredVectorQuery
	enclosed   [][]domain.ChunkID
	enclosing  map[domain.ChunkID]domain.Passage
}

func (f *nearestFake) holdsSimilarity(chunk domain.ChunkID, similarity float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.vectors == nil {
		f.vectors = map[domain.ChunkID][]byte{}
	}
	f.vectors[chunk] = storedVectorWithSimilarity(similarity)
	f.candidates = append(f.candidates, chunk)
}

func storedVectorWithSimilarity(similarity float64) []byte {
	positive := int(math.Round(dimensions * (1 + similarity) / 2))
	out := make([]byte, dimensions)
	for i := range out {
		out[i] = byte(0x9c)
		if i < positive {
			out[i] = 100
		}
	}
	return out
}

func (f *nearestFake) FindCoarseCandidates(_ context.Context, _ domain.VaultID, q source.CoarseQuery) ([]domain.ChunkID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.coarse = append(f.coarse, q)
	return f.candidates, f.coarseErr
}

func (f *nearestFake) ListStoredVectors(_ context.Context, q source.StoredVectorQuery) ([]source.StoredVector, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stored = append(f.stored, q)
	if f.storedErr != nil {
		return nil, f.storedErr
	}
	var out []source.StoredVector
	for _, id := range q.Chunks {
		if vec, ok := f.vectors[id]; ok {
			out = append(out, source.StoredVector{Chunk: id, Vector: vec})
		}
	}
	return out, nil
}

func (f *nearestFake) GetEnclosingPassages(_ context.Context, _ domain.VaultID, chunks []domain.ChunkID) ([]domain.Passage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enclosed = append(f.enclosed, chunks)
	out := make([]domain.Passage, 0, len(chunks))
	for _, id := range chunks {
		if p, ok := f.enclosing[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}
