package vectors

import (
	"cmp"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"

	indexdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/index"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vector"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// chunksPerQuery bounds the number of chunks lacking a vector returned per query.
const chunksPerQuery = 200

// Embed computes the missing vectors for the chunks of one vault.
//
// It is resumable because it holds no position: it selects the chunks that have
// no vector for the current model, so an interrupted run continues by starting
// another, and a chunk that has a vector is never selected again.
type Embed struct {
	readers vaultport.VaultReaders
	chunks  embeddingport.VectorQueries
	vectors embeddingport.VectorRepository

	Texts textdomain.Readers
	Claim vaultport.IndexClaim

	Embedder embeddingport.Embedder

	BatchCharacters int

	Concurrency int

	OnProgress func(indexdomain.EmbedResult)

	IsSkipped func(path, hash string) bool
}

func (u Embed) inferWorkers() int {
	if u.Concurrency > 0 {
		return u.Concurrency
	}
	if u.Embedder != nil && u.Embedder.Concurrency() > 0 {
		return u.Embedder.Concurrency()
	}
	return 1
}

// NewEmbed creates an Embed from the readers of vault text, the query for chunks
// lacking a vector for the current model, and the repository where vectors are
// written.
func NewEmbed(
	readers vaultport.VaultReaders, chunks embeddingport.VectorQueries, vectors embeddingport.VectorRepository,
) Embed {
	return Embed{readers: readers, chunks: chunks, vectors: vectors}
}

// EmbedWithProgress runs Execute with onProgress reporting each step when
// non-nil, and leaves without vectors the chunks of every source isSkipped
// names when non-nil.
func (u Embed) EmbedWithProgress(
	ctx context.Context, v domain.Vault, onProgress func(indexdomain.EmbedResult), isSkipped func(path, hash string) bool,
) (indexdomain.EmbedResult, error) {
	if onProgress != nil {
		u.OnProgress = onProgress
	}
	if isSkipped != nil {
		u.IsSkipped = isSkipped
	}
	return u.Execute(ctx, v)
}

// pipelineBufferSize is the number of batches buffered between pipeline stages.
const pipelineBufferSize = 4

func getSources(chunks []domain.Passage) []string {
	var out []string
	for _, one := range chunks {
		if !slices.Contains(out, one.Source) {
			out = append(out, one.Source)
		}
	}
	return out
}

// preparedBatch is one batch of chunks whose text was read and whose cached vectors were resolved.
type preparedBatch struct {
	model       embedding.Model
	pending     []domain.Passage
	hashes      [][]byte
	reused      map[int]embedding.Vector
	reusedCount int
	asking      []string
	askingMap   []int
	sources     []string
	id          uint64
}

// completedBatch is one batch whose vectors are ready to be stored.
type completedBatch struct {
	vectors []embedding.Vector
	reused  int
	id      uint64
}

// Execute embeds the chunks of a vault that lack vectors, in batches, until none remain.
func (u Embed) Execute(ctx context.Context, v domain.Vault) (res indexdomain.EmbedResult, err error) {
	if u.Embedder == nil {
		return res, nil
	}

	release, isBusy, err := u.claim(ctx)
	if isBusy {
		res.IsBusy = true
		return res, nil
	}
	if err != nil {
		return res, err
	}
	defer func() { err = errors.Join(err, release()) }()

	model := u.Embedder.GetModel()
	reader, err := u.readers.Open(v)
	if err != nil {
		return res, err
	}
	source := extracted{files: reader}
	if u.Texts != nil {
		if source.texts, err = u.Texts.Open(v); err != nil {
			return res, err
		}
	}

	var mu sync.Mutex
	workers := max(u.inferWorkers(), 1)
	track := newTracker(u, workers, &res, &mu)
	updateProgress := func(comp completedBatch) {
		track.leave(comp.id)
		track.publish(func(r *indexdomain.EmbedResult) {
			r.Embedded += len(comp.vectors)
			r.Reused += comp.reused
		})
	}

	g, gctx := errgroup.WithContext(ctx)
	bufferSize := max(pipelineBufferSize, workers*2)
	preparedChan := make(chan preparedBatch, bufferSize)
	completedChan := make(chan completedBatch, bufferSize)

	g.Go(func() error {
		defer close(preparedChan)
		return u.produceBatches(gctx, v, model, &source, track, preparedChan)
	})
	g.Go(func() error {
		defer close(completedChan)
		return u.inferBatches(gctx, workers, preparedChan, completedChan)
	})
	g.Go(func() error {
		return u.storeBatches(gctx, completedChan, updateProgress)
	})

	if err := g.Wait(); err != nil {
		return res, err
	}
	return res, nil
}

func (u Embed) claim(ctx context.Context) (func() error, bool, error) {
	if u.Claim == nil {
		return func() error { return nil }, false, nil
	}
	release, err := u.Claim.ClaimIndex(ctx)
	if errors.Is(err, vault.ErrClaimed) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return release, false, nil
}

func (u Embed) produceBatches(
	ctx context.Context,
	v domain.Vault,
	model embedding.Model,
	source *extracted,
	track *tracker,
	out chan<- preparedBatch,
) error {
	res, mu := track.res, track.mu
	var after embedding.ChunkCursor
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		pending, next, err := u.chunks.GetUnembeddedChunks(ctx, v.ID, model, after, chunksPerQuery)
		if err != nil {
			return fmt.Errorf("pending vectors from %s: %w", model, err)
		}
		if len(pending) == 0 {
			return nil
		}
		if next == after {
			return fmt.Errorf("pending vectors from %s do not continue past %d chunks", model, len(pending))
		}
		mu.Lock()
		res.Pending += len(pending)
		mu.Unlock()
		after = next

		chunks, texts, err := u.read(ctx, source, pending, res, mu, track)
		if err != nil {
			return err
		}
		sortBySource(chunks, texts)
		if err := u.sendBatches(ctx, model, chunks, texts, track, out); err != nil {
			return err
		}
	}
}

func (u Embed) sendBatches(
	ctx context.Context,
	model embedding.Model,
	chunks []domain.Passage,
	texts []string,
	track *tracker,
	out chan<- preparedBatch,
) error {
	pos := 0
	for _, batch := range splitBatches(texts, u.BatchCharacters) {
		prep, err := u.prepareBatch(ctx, model, chunks[pos:pos+len(batch)], batch)
		if err != nil {
			return err
		}
		pos += len(batch)
		prep.id = track.enter(prep.sources)
		// The batch is reported before it is sent, so no worker embeds it before a report names it.
		track.publish(func(*indexdomain.EmbedResult) {})
		select {
		case <-ctx.Done():
			track.withdraw(prep.id)
			return ctx.Err()
		case out <- prep:
		}
	}
	return nil
}

func sortBySource(chunks []domain.Passage, texts []string) {
	if len(chunks) < 2 {
		return
	}
	type passageItem struct {
		chunk       domain.Passage
		text        string
		sourceIndex int
	}
	sourceOrder := make(map[string]int)
	for _, c := range chunks {
		if _, ok := sourceOrder[c.Source]; !ok {
			sourceOrder[c.Source] = len(sourceOrder)
		}
	}
	items := make([]passageItem, len(chunks))
	for i := range chunks {
		items[i] = passageItem{
			chunk:       chunks[i],
			text:        texts[i],
			sourceIndex: sourceOrder[chunks[i].Source],
		}
	}
	slices.SortStableFunc(items, func(a, b passageItem) int {
		if a.sourceIndex != b.sourceIndex {
			return cmp.Compare(a.sourceIndex, b.sourceIndex)
		}
		return cmp.Compare(len(a.text), len(b.text))
	})
	for i, item := range items {
		chunks[i] = item.chunk
		texts[i] = item.text
	}
}

func (u Embed) inferBatches(
	ctx context.Context,
	workers int,
	in <-chan preparedBatch,
	out chan<- completedBatch,
) error {
	g, gctx := errgroup.WithContext(ctx)
	for range workers {
		g.Go(func() error { return u.runInferWorker(gctx, in, out) })
	}
	return g.Wait()
}

