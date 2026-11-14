package vectors

import (
	"context"
	"hash/fnv"
	"math/rand/v2"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfake"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext"

	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// embedder returns the same vector for the same text, so a returned vector
// identifies its input.
type embedder struct {
	dims   int
	minLen int

	mu        sync.Mutex
	calls     int
	seenCount []string
}

func (e *embedder) Describe() searchsource.Description {
	return searchsource.Description{Name: "fake", Width: e.dims, MinTextLength: e.minLen}
}

func (*embedder) Close() error { return nil }

// interruptedStore cancels the run after its Nth write and rejects every later
// write, so exactly N groups of vectors are stored.
type interruptedStore struct {
	*indexfake.Store
	cancel context.CancelFunc
	after  int
	saved  int
}

func (s *interruptedStore) SaveVectors(ctx context.Context, vectors []vector.Stored) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.Store.SaveVectors(ctx, vectors); err != nil {
		return err
	}
	s.saved++
	if s.saved == s.after {
		s.cancel()
	}
	return nil
}

func (e *embedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	e.mu.Lock()
	e.calls++
	e.mu.Unlock()
	out := make([][]float32, 0, len(texts))
	for _, text := range texts {
		e.mu.Lock()
		e.seenCount = append(e.seenCount, text)
		e.mu.Unlock()
		out = append(out, vectorOf(text, e.dims))
	}
	return out, nil
}

func vectorOf(text string, dims int) []float32 {
	sum := fnv.New64a()
	sum.Write([]byte(text))
	numbers := rand.New(rand.NewPCG(sum.Sum64(), 0x9E3779B97F4A7C15))

	out := make([]float32, dims)
	for i := range out {
		out[i] = float32(numbers.NormFloat64())
	}
	return out
}

func newEmbed(readers vaultport.VaultReaders, chunks searchsourceport.VectorQueries, vectors searchsourceport.VectorRepository) Embed {
	return NewEmbed(readers, chunks, vectors, openClaim{}, bookTexts{readers: readers}, noModel{})
}

func newExtract(
	readers vaultport.VaultReaders, index *indexfake.Store,
) filetext.Index {
	extract := filetext.NewIndex(filetext.IndexDeps{
		Readers: readers, Sources: index, Records: index, Backlog: index, Lookup: index,
		Texts: bookTexts{readers: readers},
	})
	return extract
}

// openClaim holds nothing and is never busy.
type openClaim struct{}

func (openClaim) ClaimIndex(context.Context) (func() error, error) {
	return func() error { return nil }, nil
}

// noModel describes no model, so nothing is embedded until a test sets a source.
type noModel struct{}

func (noModel) Describe() searchsource.Description { return searchsource.Description{} }

func (noModel) IndexTexts(context.Context, searchsource.Place, []searchsource.Text) ([][]float32, error) {
	return nil, nil
}

func (noModel) SearchPassages(context.Context, searchsource.Place, searchsource.Query) ([]searchsource.Match, error) {
	return nil, nil
}
