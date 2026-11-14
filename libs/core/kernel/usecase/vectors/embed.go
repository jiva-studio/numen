package vectors

import (
	"context"
	"errors"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
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

	texts  textdomain.Readers
	claim  vaultport.IndexClaim
	source searchsourceport.Source

	BatchCharacters int

	Concurrency int
}

func (u Embed) inferWorkers() int {
	if u.Concurrency > 0 {
		return u.Concurrency
	}
	if u.source.Describe().Concurrency > 0 {
		return u.source.Describe().Concurrency
	}
	return 1
}

// NewEmbed creates an Embed from the readers of vault text, the query for chunks
// lacking a vector for the current model, the repository where vectors are
// written, the claim on the index, the readers of file texts and the search source.
// A nil claim holds nothing, nil texts means no file has text, and a nil source embeds nothing.
func NewEmbed(
	readers vaultport.VaultReaders, chunks searchsourceport.VectorQueries, vectors searchsourceport.VectorRepository,
	claim vaultport.IndexClaim, texts textdomain.Readers, source searchsourceport.Source,
) Embed {
	if claim == nil {
		claim = noClaim{}
	}
	if texts == nil {
		texts = noTextReaders{}
	}
	if source == nil {
		source = noSource{}
	}
	return Embed{readers: readers, chunks: chunks, vectors: vectors, claim: claim, texts: texts, source: source}
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
	ctx = plugin.WithBackground(ctx, plugin.WhoIndexing)
	model := u.source.Describe()
	if !model.IsDescribed() {
		return res, nil
	}

	release, isBusy, err := u.claimIndex(ctx)
	if isBusy {
		res.IsBusy = true
		return res, nil
	}
	if err != nil {
		return res, err
	}
	defer func() { err = errors.Join(err, release()) }()

	reader, err := u.readers.Open(v)
	if err != nil {
		return res, err
	}
	source := extracted{files: reader}
	if source.texts, err = u.texts.Open(v); err != nil {
		return res, err
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
		return inferrer{source: u.source}.inferBatches(gctx, workers, preparedChan, completedChan)
	})
	g.Go(func() error {
		return storer{vectors: u.vectors}.storeBatches(gctx, completedChan, updateProgress)
	})

	if err := g.Wait(); err != nil {
		return res, err
	}
	return res, nil
}

func (u Embed) claimIndex(ctx context.Context) (func() error, bool, error) {
	release, err := u.claim.ClaimIndex(ctx)
	if errors.Is(err, vault.ErrClaimed) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return release, false, nil
}

// noClaim holds nothing and is never busy.
type noClaim struct{}

func (noClaim) ClaimIndex(context.Context) (func() error, error) {
	return func() error { return nil }, nil
}

// noTextReaders is a vault where no file has text.
type noTextReaders struct{}

func (noTextReaders) Open(domain.Vault) (textdomain.VaultTexts, error) { return nil, nil }

// noSource is the search source of an installation that has none: it describes nothing, so nothing is embedded.
type noSource struct{}

func (noSource) Describe() searchsource.Description { return searchsource.Description{} }

func (noSource) IndexTexts(context.Context, searchsource.Place, []searchsource.Text) ([][]float32, error) {
	return nil, nil
}

func (noSource) SearchPassages(context.Context, searchsource.Place, searchsource.Query) ([]searchsource.Match, error) {
	return nil, nil
}
