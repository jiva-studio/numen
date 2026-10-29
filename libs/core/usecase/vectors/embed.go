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

	"github.com/jiva-studio/numen/libs/core/internal/vaultreading"
	"github.com/jiva-studio/numen/libs/core/internal/vector"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
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

	// Texts is optional. A chunk of a source cut from a text version is re-sliced
	// out of that version; without it such a chunk is left owing its vector.
	Texts textport.TextReaders
	// Claim is optional. It keeps a second process from embedding the vault at
	// once; without it nothing is claimed.
	Claim vaultport.IndexClaim

	// Embedder is optional. Without one nothing is embedded and a search answers
	// on its words alone, which is a whole search: the vector index fills in
	// behind, and may never finish.
	Embedder embeddingport.Embedder

	// BatchCharacters bounds one request to the model. Zero takes the default.
	BatchCharacters int

	// Concurrency is how many infer workers run in parallel. Zero takes the
	// embedder's concurrency.
	Concurrency int

	// OnProgress, if set, is called each time a group of vectors is written.
	OnProgress func(EmbedResult)
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

// EmbedResult reports what embedding did.
type EmbedResult = vaultreading.EmbedResult

// EmbedWithProgress runs Execute with onProgress reporting each step when non-nil.
func (u Embed) EmbedWithProgress(ctx context.Context, v domain.Vault, onProgress func(EmbedResult)) (EmbedResult, error) {
	if onProgress != nil {
		u.OnProgress = onProgress
	}
	return u.Execute(ctx, v)
}

// pipelineBufferSize is the number of batches buffered between pipeline stages.
const pipelineBufferSize = 4

// preparedBatch is one batch of chunks whose text was read and whose cached vectors were resolved.
type preparedBatch struct {
	model       embeddingport.EmbeddingModel
	owing       []domain.Passage
	hashes      [][]byte
	reused      map[int]embeddingport.Vector
	reusedCount int
	asking      []string
	askingMap   []int
	source      string
}

// completedBatch is one batch whose vectors are ready to be stored.
type completedBatch struct {
	vectors []embeddingport.Vector
	reused  int
	source  string
}

// Execute embeds the chunks of a vault that lack vectors, in batches, until none remain.
func (u Embed) Execute(ctx context.Context, v domain.Vault) (EmbedResult, error) {
	var res EmbedResult
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
	defer func() { _ = release() }()

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
	updateProgress := func(source string, embedded, reused int) {
		mu.Lock()
		if source != "" {
			res.CurrentSource = source
		}
		res.Embedded += embedded
		res.Reused += reused
		snapshot := res
		mu.Unlock()
		u.progress(snapshot)
	}

	workers := max(u.inferWorkers(), 1)

	g, gctx := errgroup.WithContext(ctx)
	bufferSize := max(pipelineBufferSize, workers*2)
	preparedChan := make(chan preparedBatch, bufferSize)
	completedChan := make(chan completedBatch, bufferSize)

	g.Go(func() error {
		defer close(preparedChan)
		return u.produceBatches(gctx, v, model, &source, &res, &mu, preparedChan)
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

// claim takes the index claim when one is set, and reports isBusy when
// another process holds it.
func (u Embed) claim(ctx context.Context) (func() error, bool, error) {
	if u.Claim == nil {
		return func() error { return nil }, false, nil
	}
	release, err := u.Claim.ClaimIndex(ctx)
	if errors.Is(err, vaultport.ErrClaimed) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return release, false, nil
}

// produceBatches reads every owing chunk of the vault and sends them on out as
// prepared batches.
func (u Embed) produceBatches(
	ctx context.Context,
	v domain.Vault,
	model embeddingport.EmbeddingModel,
	source *extracted,
	res *EmbedResult,
	mu *sync.Mutex,
	out chan<- preparedBatch,
) error {
	var after embeddingport.ChunkCursor
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		owing, next, err := u.chunks.GetUnembeddedChunks(ctx, v.ID, model, after, chunksPerQuery)
		if err != nil {
			return fmt.Errorf("pending vectors from %s: %w", model, err)
		}
		if len(owing) == 0 {
			return nil
		}
		if next == after {
			return fmt.Errorf("pending vectors from %s do not continue past %d chunks", model, len(owing))
		}
		mu.Lock()
		res.Owing += len(owing)
		mu.Unlock()
		after = next

		chunks, texts, err := u.read(ctx, source, owing, res, mu)
		if err != nil {
			return err
		}
		sortBySource(chunks, texts)
		if err := u.sendBatches(ctx, model, chunks, texts, out); err != nil {
			return err
		}
	}
}

// sendBatches splits the texts into batches by size and sends each one prepared.
func (u Embed) sendBatches(
	ctx context.Context,
	model embeddingport.EmbeddingModel,
	chunks []domain.Passage,
	texts []string,
	out chan<- preparedBatch,
) error {
	pos := 0
	for _, batch := range splitBatches(texts, u.BatchCharacters) {
		prep, err := u.prepareBatch(ctx, model, chunks[pos:pos+len(batch)], batch)
		if err != nil {
			return err
		}
		pos += len(batch)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- prep:
		}
	}
	return nil
}

// sortBySource orders chunks and their texts in place: sources in order of first
// appearance, and within a source by ascending text length.
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

// inferBatches runs workers that turn prepared batches into completed ones.
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

// runInferWorker infers batches from in until it closes or ctx ends.
func (u Embed) runInferWorker(ctx context.Context, in <-chan preparedBatch, out chan<- completedBatch) error {
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

// storeBatches writes the vectors of each completed batch and reports progress.
func (u Embed) storeBatches(
	ctx context.Context,
	in <-chan completedBatch,
	updateProgress func(source string, embedded, reused int),
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case comp, ok := <-in:
			if !ok {
				return nil
			}
			if len(comp.vectors) > 0 {
				if err := u.vectors.SaveVectors(ctx, comp.vectors); err != nil {
					return fmt.Errorf("write %d vectors: %w", len(comp.vectors), err)
				}
			}
			updateProgress(comp.source, len(comp.vectors), comp.reused)
		}
	}
}

// read recovers the text of each chunk by reopening its source; the index stores
// chunk positions, not content.
//
// A chunk whose source is gone, or whose position is not in the current source
// text, is left unchanged: the file is the source of truth, and the index
// follows it when the file is next read.
func (u Embed) read(
	ctx context.Context,
	source *extracted,
	owing []domain.Passage,
	res *EmbedResult,
	mu *sync.Mutex,
) ([]domain.Passage, []string, error) {
	chunks := make([]domain.Passage, 0, len(owing))
	texts := make([]string, 0, len(owing))

	for _, p := range owing {
		mu.Lock()
		if p.Source != res.CurrentSource {
			res.CurrentSource = p.Source
			snapshot := *res
			mu.Unlock()
			u.progress(snapshot)
		} else {
			mu.Unlock()
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

// prepareBatch resolves cached vectors and identifies texts requiring inference.
func (u Embed) prepareBatch(
	ctx context.Context,
	model embeddingport.EmbeddingModel,
	owing []domain.Passage,
	texts []string,
) (preparedBatch, error) {
	if len(texts) == 0 {
		return preparedBatch{}, nil
	}
	recipe := model.Recipe()

	hashes := make([][]byte, len(texts))
	for i := range texts {
		raw, err := hex.DecodeString(owing[i].ChunkHash)
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
		owing:     owing,
		hashes:    hashes,
		reused:    make(map[int]embeddingport.Vector),
		asking:    make([]string, 0, len(texts)),
		askingMap: make([]int, len(texts)),
		source:    owing[len(owing)-1].Source,
	}
	var deduplicated map[string]int
	if len(texts) > 1 {
		deduplicated = make(map[string]int, len(texts))
	}
	for i := range texts {
		value, exists := retained[hex.EncodeToString(hashes[i])]
		if !exists || len(hashes[i]) == 0 {
			key := owing[i].ChunkHash
			if key == "" {
				key = texts[i]
			}
			prep.queueText(i, key, texts[i], deduplicated)
			continue
		}
		prep.askingMap[i] = -1
		prep.reused[i] = embeddingport.Vector{
			ChunkID: owing[i].ChunkID,
			Hash:    hashes[i],
			Model:   model,
			Kind:    embeddingport.QuantisedInt8,
			Value:   value,
			Coarse:  vector.Coarse(vector.Dimensions(value)),
		}
		prep.reusedCount++
	}
	return prep, nil
}

// queueText queues the text at index i for inference. A nil visited map queues
// every text; otherwise a key already in visited shares the position of its
// first text.
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

// inferBatch embeds texts that owe vectors and normalises/quantises the output.
func (u Embed) inferBatch(ctx context.Context, prep preparedBatch) (completedBatch, error) {
	if len(prep.owing) == 0 {
		return completedBatch{}, nil
	}
	var unique []uniqueVector
	if len(prep.asking) > 0 {
		vectors, err := u.Embedder.Embed(ctx, prep.asking)
		if err != nil {
			return completedBatch{}, fmt.Errorf("embed %d chunks: %w", len(prep.asking), err)
		}
		if len(vectors) != len(prep.asking) {
			return completedBatch{}, fmt.Errorf("%s answered with %d vectors for %d texts", prep.model, len(vectors), len(prep.asking))
		}
		unique = make([]uniqueVector, len(vectors))
		for i, v := range vectors {
			if len(v) != prep.model.Dimensions {
				return completedBatch{}, fmt.Errorf("%s answered with %d dimensions", prep.model, len(v))
			}
			v = vector.Normalise(v)
			quantised := vector.Bytes(v)
			unique[i] = uniqueVector{
				value:  packVector(quantised),
				coarse: vector.Coarse(quantised),
			}
		}
	}

	out := make([]embeddingport.Vector, len(prep.owing))
	for i := range prep.owing {
		if pos := prep.askingMap[i]; pos >= 0 {
			out[i] = embeddingport.Vector{
				ChunkID: prep.owing[i].ChunkID,
				Hash:    prep.hashes[i],
				Model:   prep.model,
				Kind:    embeddingport.QuantisedInt8,
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
		source:  prep.source,
	}, nil
}

func (u Embed) progress(res EmbedResult) {
	if u.OnProgress != nil {
		u.OnProgress(res)
	}
}

// packVector is a quantised vector as the bytes that are stored, one per
// dimension.
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
	texts     textport.Texts
	path      string
	text      string
	isIndexed bool
}

// textOf returns the extracted text of one source, and false when the vault no
// longer holds a source with text at that path.
//
// A note chunk is sliced from the note's file, and a version chunk from that
// version.
func (e *extracted) textOf(ctx context.Context, path, from string) (string, bool, error) {
	if e.path == path {
		return e.text, e.isIndexed, nil
	}
	e.path, e.text, e.isIndexed = path, "", false

	read, err := e.read(ctx, path, from)
	switch {
	case vaultport.IsNoNote(err), errors.Is(err, vaultport.ErrNotExist), errors.Is(err, textport.ErrNoText):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("read %s: %w", path, err)
	}
	e.text = read
	e.isIndexed = true
	return e.text, true, nil
}

// read is the text a source's chunks are places in: a note's file, or the
// version the source was cut from.
func (e *extracted) read(ctx context.Context, path, from string) (string, error) {
	if from == "" {
		raw, err := e.files.Read(ctx, path)
		return string(raw), err
	}
	if e.texts == nil {
		return "", textport.ErrNoText
	}
	doc, err := e.texts.GetVersionText(ctx, path, domain.TextVersionID(from))
	return doc.Text, err
}
