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

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vector"
	"github.com/jiva-studio/numen/libs/note/domain"

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

	OnProgress func(vaultreading.EmbedResult)

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
	ctx context.Context, v domain.Vault, onProgress func(vaultreading.EmbedResult), isSkipped func(path, hash string) bool,
) (vaultreading.EmbedResult, error) {
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

// producer reads the chunks that lack vectors and cuts them into batches with the vectors already kept.
type producer struct {
	chunks          embeddingport.VectorQueries
	vectors         embeddingport.VectorRepository
	batchCharacters int
	isSkipped       func(path, hash string) bool
}

func newProducer(u Embed) producer {
	return producer{chunks: u.chunks, vectors: u.vectors, batchCharacters: u.BatchCharacters, isSkipped: u.IsSkipped}
}

// inferrer embeds the texts of the batches the producer cut.
type inferrer struct {
	embedder embeddingport.Embedder
}

// storer writes the vectors of the completed batches.
type storer struct {
	vectors embeddingport.VectorRepository
}

// Execute embeds the chunks of a vault that lack vectors, in batches, until none remain.
func (u Embed) Execute(ctx context.Context, v domain.Vault) (res vaultreading.EmbedResult, err error) {
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
	track := newTracker(u.OnProgress, workers, &res, &mu)
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
		return newProducer(u).produceBatches(gctx, v, model, &source, track, preparedChan)
	})
	g.Go(func() error {
		defer close(completedChan)
		return inferrer{embedder: u.Embedder}.inferBatches(gctx, workers, preparedChan, completedChan)
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

func (u producer) produceBatches(
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

func (u producer) sendBatches(
	ctx context.Context,
	model embedding.Model,
	chunks []domain.Passage,
	texts []string,
	track *tracker,
	out chan<- preparedBatch,
) error {
	pos := 0
	for _, batch := range splitBatches(texts, u.batchCharacters) {
		prep, err := u.prepareBatch(ctx, model, chunks[pos:pos+len(batch)], batch)
		if err != nil {
			return err
		}
		pos += len(batch)
		prep.id = track.enter(prep.sources)
		// The batch is reported before it is sent, so no worker embeds it before a report names it.
		track.publish(func(*vaultreading.EmbedResult) {})
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

func (u inferrer) inferBatches(
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

func (u inferrer) runInferWorker(ctx context.Context, in <-chan preparedBatch, out chan<- completedBatch) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case prep, ok := <-in:
			if !ok {
				return nil
			}
			completed, err := u.inferBatch(ctx, prep)
			if err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- completed:
			}
		}
	}
}

func (u storer) storeBatches(
	ctx context.Context,
	in <-chan completedBatch,
	updateProgress func(comp completedBatch),
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case comp, ok := <-in:
			if !ok {
				return nil
			}
			if err := u.saveCompleted(ctx, comp); err != nil {
				return err
			}
			updateProgress(comp)
		}
	}
}

func (u storer) saveCompleted(ctx context.Context, comp completedBatch) error {
	if len(comp.vectors) == 0 {
		return nil
	}
	if err := u.vectors.SaveVectors(ctx, comp.vectors); err != nil {
		return fmt.Errorf("write %d vectors: %w", len(comp.vectors), err)
	}
	return nil
}

func (u producer) read(
	ctx context.Context,
	source *extracted,
	pending []domain.Passage,
	res *vaultreading.EmbedResult,
	mu *sync.Mutex,
	track *tracker,
) ([]domain.Passage, []string, error) {
	chunks := make([]domain.Passage, 0, len(pending))
	texts := make([]string, 0, len(pending))

	isFirst := true
	for _, p := range pending {
		if u.isSkipped != nil && u.isSkipped(p.Source, p.ContentHash) {
			mu.Lock()
			res.Skipped++
			mu.Unlock()
			continue
		}
		if isFirst {
			isFirst = false
			track.publishNext(p.Source)
		}
		prose, ok, err := source.textOf(ctx, p.Source, p.Producer)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			mu.Lock()
			res.Vanished++
			mu.Unlock()
			continue
		}
		if p.Start < 0 || p.Length <= 0 || p.Start+p.Length > len(prose) {
			mu.Lock()
			res.Displaced++
			mu.Unlock()
			continue
		}
		chunks = append(chunks, p)
		texts = append(texts, prose[p.Start:p.Start+p.Length])
	}
	return chunks, texts, nil
}

func (u producer) prepareBatch(
	ctx context.Context,
	model embedding.Model,
	pending []domain.Passage,
	texts []string,
) (preparedBatch, error) {
	if len(texts) == 0 {
		return preparedBatch{}, nil
	}
	recipe := model.Recipe()

	hashes := make([][]byte, len(texts))
	for i := range texts {
		raw, err := hex.DecodeString(pending[i].ChunkHash)
		if err != nil || len(raw) == 0 {
			continue
		}
		hashes[i] = raw
	}
	retained, err := u.vectors.GetKeptVectors(ctx, recipe, hashes)
	if err != nil {
		return preparedBatch{}, fmt.Errorf("what is already made: %w", err)
	}

	prep := preparedBatch{
		model:     model,
		pending:   pending,
		hashes:    hashes,
		reused:    make(map[int]embedding.Vector),
		asking:    make([]string, 0, len(texts)),
		askingMap: make([]int, len(texts)),
		sources:   getSources(pending),
	}
	var deduplicated map[string]int
	if len(texts) > 1 {
		deduplicated = make(map[string]int, len(texts))
	}
	for i := range texts {
		value, exists := retained[hex.EncodeToString(hashes[i])]
		if !exists || len(hashes[i]) == 0 {
			key := pending[i].ChunkHash
			if key == "" {
				key = texts[i]
			}
			prep.queueText(i, key, texts[i], deduplicated)
			continue
		}
		prep.askingMap[i] = -1
		prep.reused[i] = embedding.Vector{
			ChunkID: pending[i].ChunkID,
			Hash:    hashes[i],
			Model:   model,
			Kind:    embedding.QuantizedInt8,
			Value:   value,
			Coarse:  vector.Coarse(vector.Dimensions(value)),
		}
		prep.reusedCount++
	}
	return prep, nil
}

func (p *preparedBatch) queueText(i int, key, value string, visited map[string]int) {
	if visited != nil {
		if pos, ok := visited[key]; ok {
			p.askingMap[i] = pos
			return
		}
		visited[key] = len(p.asking)
	}
	p.askingMap[i] = len(p.asking)
	p.asking = append(p.asking, value)
}

type uniqueVector struct {
	value  []byte
	coarse []byte
}

func (u inferrer) inferBatch(ctx context.Context, prep preparedBatch) (completedBatch, error) {
	if len(prep.pending) == 0 {
		return completedBatch{}, nil
	}
	unique, err := u.embedAsking(ctx, prep)
	if err != nil {
		return completedBatch{}, err
	}

	out := make([]embedding.Vector, len(prep.pending))
	for i := range prep.pending {
		if pos := prep.askingMap[i]; pos >= 0 {
			out[i] = embedding.Vector{
				ChunkID: prep.pending[i].ChunkID,
				Hash:    prep.hashes[i],
				Model:   prep.model,
				Kind:    embedding.QuantizedInt8,
				Value:   unique[pos].value,
				Coarse:  unique[pos].coarse,
			}
		} else if v, ok := prep.reused[i]; ok {
			out[i] = v
		}
	}
	return completedBatch{
		vectors: out,
		reused:  prep.reusedCount,
		id:      prep.id,
	}, nil
}

func packVector(q []int8) []byte {
	out := make([]byte, len(q))
	for i, x := range q {
		out[i] = byte(x)
	}
	return out
}

// extracted is the text of one source, to which chunk offsets refer, read from
// the source again. The most recently opened source is cached, because the
// chunks of one source are processed together.
type extracted struct {
	files     vaultport.VaultReader
	texts     textdomain.Texts
	path      string
	text      string
	isIndexed bool
}

func (e *extracted) textOf(ctx context.Context, path, from string) (string, bool, error) {
	if e.path == path {
		return e.text, e.isIndexed, nil
	}
	e.path, e.text, e.isIndexed = path, "", false

	read, err := e.read(ctx, path, from)
	switch {
	case vaultport.IsNoNote(err), errors.Is(err, vaultport.ErrNotExist), errors.Is(err, textdomain.ErrNoText):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("read %s: %w", path, err)
	}
	e.text = read
	e.isIndexed = true
	return e.text, true, nil
}

func (e *extracted) read(ctx context.Context, path, from string) (string, error) {
	if from == "" {
		raw, err := e.files.Read(ctx, path)
		return string(raw), err
	}
	if e.texts == nil {
		return "", textdomain.ErrNoText
	}
	doc, err := e.texts.GetVersionText(ctx, path, domain.TextVersionID(from))
	return doc.Text, err
}

func (u inferrer) embedAsking(ctx context.Context, prep preparedBatch) ([]uniqueVector, error) {
	if len(prep.asking) == 0 {
		return nil, nil
	}
	vectors, err := u.embedder.Embed(ctx, prep.asking)
	if err != nil {
		return nil, fmt.Errorf("embed %d chunks: %w", len(prep.asking), err)
	}
	if len(vectors) != len(prep.asking) {
		return nil, fmt.Errorf("%s answered with %d vectors for %d texts", prep.model, len(vectors), len(prep.asking))
	}
	unique := make([]uniqueVector, len(vectors))
	for i, v := range vectors {
		if len(v) != prep.model.Dimensions {
			return nil, fmt.Errorf("%s answered with %d dimensions", prep.model, len(v))
		}
		quantized := vector.Bytes(vector.Normalize(v))
		unique[i] = uniqueVector{
			value:  packVector(quantized),
			coarse: vector.Coarse(quantized),
		}
	}
	return unique, nil
}
