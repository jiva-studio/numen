package vectors

import (
	"context"
	"errors"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/note/domain"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// chunksPerQuery bounds the number of chunks lacking a vector returned per query.
const chunksPerQuery = 200

// Embed hands the chunks of one vault that have no vector to the search source, and keeps the vectors it answers.
//
// It is resumable because it holds no position: it selects the chunks that have
// no vector for the current model, so an interrupted run continues by starting
// another, and a chunk that has a vector is never selected again.
type Embed struct {
	readers vaultport.VaultReaders
	chunks  searchsourceport.VectorQueries
	vectors searchsourceport.VectorRepository

	Texts textdomain.Readers
	Claim vaultport.IndexClaim

	Source searchsourceport.Source

	BatchCharacters int

	Concurrency int
}

func (u Embed) inferWorkers() int {
	if u.Concurrency > 0 {
		return u.Concurrency
	}
	if u.Source != nil && u.Source.Describe().Concurrency > 0 {
		return u.Source.Describe().Concurrency
	}
	return 1
}

// NewEmbed creates an Embed from the readers of vault text, the query for chunks
// lacking a vector for the current model, and the repository where vectors are
// written.
func NewEmbed(
	readers vaultport.VaultReaders, chunks searchsourceport.VectorQueries, vectors searchsourceport.VectorRepository,
) Embed {
	return Embed{readers: readers, chunks: chunks, vectors: vectors}
}

// EmbedWithProgress runs Execute with onProgress reporting each step when
// non-nil, and leaves without vectors the chunks of every source isSkipped
// names when non-nil.
func (u Embed) EmbedWithProgress(
	ctx context.Context, v domain.Vault, onProgress func(vaultreading.EmbedResult), isSkipped func(path, hash string) bool,
) (vaultreading.EmbedResult, error) {
	return u.run(ctx, v, onProgress, isSkipped)
}

// pipelineBufferSize is the number of batches buffered between pipeline stages.
const pipelineBufferSize = 4

// Execute embeds the chunks of a vault that lack vectors, in batches, until none remain.
func (u Embed) Execute(ctx context.Context, v domain.Vault) (vaultreading.EmbedResult, error) {
	return u.run(ctx, v, nil, nil)
}

func (u Embed) run(
	ctx context.Context, v domain.Vault, onProgress func(vaultreading.EmbedResult), isSkipped func(path, hash string) bool,
) (res vaultreading.EmbedResult, err error) {
	if u.Source == nil {
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

	model := u.Source.Describe()
	if !model.IsDescribed() {
		return res, nil
	}
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
	track := newTracker(onProgress, workers, &res, &mu)
	updateProgress := func(comp completedBatch) {
		track.leave(comp.id)
		track.publish(func(r *vaultreading.EmbedResult) {
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
		return newProducer(u, isSkipped).produceBatches(gctx, v, model, &source, track, preparedChan)
	})
	g.Go(func() error {
		defer close(completedChan)
		return inferrer{source: u.Source}.inferBatches(gctx, workers, preparedChan, completedChan)
	})
	g.Go(func() error {
		return storer{vectors: u.vectors}.storeBatches(gctx, completedChan, updateProgress)
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
