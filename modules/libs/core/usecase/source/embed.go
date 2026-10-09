package source

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// chunksPerQuery bounds one answer about what owes a vector.
const chunksPerQuery = 200

// Embed gives the chunks of one vault the vectors they owe.
//
// It resumes because it holds no position of its own: what to do next is
// whatever has no vector from the model in use, so a run that was interrupted is
// continued by starting another, and a chunk that has one is never asked about
// again.
type Embed struct {
	readers port.VaultReaders
	chunks  port.VectorQueries
	vectors port.VectorRepository

	// Derived is optional. It holds what a recogniser wrote, and a chunk of a
	// recognised document is re-sliced out of that and not out of the document;
	// without one such a chunk is left owing its vector.
	Derived port.DerivedStore
	// Documents is optional. It reads a format that needs a library, for a
	// source standing on its own bytes; without one such a chunk is left owing
	// its vector.
	Documents port.TextExtractor

	// Embedder is optional. Without one nothing is embedded and a search answers
	// on its words alone, which is a whole search: the vector index fills in
	// behind, and may never finish.
	Embedder port.Embedder

	// BatchCharacters bounds one request to the model. Zero takes the default.
	BatchCharacters int

	// OnProgress, if set, is called each time a group of vectors is written.
	OnProgress func(EmbedResult)
}

// NewEmbed is what a vault's chunks are given vectors through: the vault the
// text is read out of, what says which chunks owe a vector from the model in
// use, and where the vectors are written.
func NewEmbed(
	readers port.VaultReaders, chunks port.VectorQueries, vectors port.VectorRepository,
) Embed {
	return Embed{readers: readers, chunks: chunks, vectors: vectors}
}

// EmbedResult reports what embedding did.
type EmbedResult struct {
	Owing    int // chunks found with no vector from the model in use
	Embedded int // chunks that now carry one
	// Reused counts the chunks whose vector was already made for their text and
	// was claimed rather than bought.
	Reused    int
	Vanished  int    // whose source the vault no longer holds
	Displaced int    // whose place is not in the text their source holds now
	Reading   string // the source open now
	IsBusy    bool   // somebody else is embedding this vault, and nothing was done
}

// pipelineBufferSize is the number of batches buffered between pipeline stages.
const pipelineBufferSize = 4

// preparedBatch is one batch of chunks whose text was read and whose cached vectors were resolved.
type preparedBatch struct {
	model       port.EmbeddingModel
	owing       []domain.Passage
	hashes      [][]byte
	reused      []port.Vector
	reusedCount int
	asking      []string
	askingMap   []int
	reading     string
}

// completedBatch is one batch whose vectors are ready to be stored.
type completedBatch struct {
	vectors []port.Vector
	reused  int
	reading string
}

// Execute embeds what one vault owes, in groups, until nothing owes anything.
func (u Embed) Execute(ctx context.Context, v domain.Vault) (EmbedResult, error) {
	var res EmbedResult
	if u.Embedder == nil {
		return res, nil
	}

	if u.Derived != nil {
		release, err := u.Derived.Claim(ctx, text.IndexClaim)
		if errors.Is(err, port.ErrClaimed) {
			res.IsBusy = true
			return res, nil
		}
		if err != nil {
			return res, err
		}
		defer release()
	}

	model := u.Embedder.Model()
	reader, err := u.readers.Open(v)
	if err != nil {
		return res, err
	}
	var store port.DerivedStore
	if u.Derived != nil {
		store = u.Derived
	}
	source := extracted{of: text.Reader{Vault: reader, Derived: store, Documents: u.Documents}}

	var mu sync.Mutex
	updateProgress := func(reading string, embedded, reused int) {
		mu.Lock()
		if reading != "" {
			res.Reading = reading
		}
		res.Embedded += embedded
		res.Reused += reused
		snapshot := res
		mu.Unlock()
		u.progress(snapshot)
	}

	g, gctx := errgroup.WithContext(ctx)
	preparedChan := make(chan preparedBatch, pipelineBufferSize)
	completedChan := make(chan completedBatch, pipelineBufferSize)

	g.Go(func() error {
		defer close(preparedChan)
		var after port.ChunkCursor
		for {
			if err := gctx.Err(); err != nil {
				return err
			}
			owing, next, err := u.chunks.GetUnembeddedChunks(gctx, v.ID, model, after, chunksPerQuery)
			if err != nil {
				return fmt.Errorf("what owes a vector from %s: %w", model, err)
			}
			if len(owing) == 0 {
				return nil
			}
			if next == after {
				return fmt.Errorf("what owes a vector from %s does not carry on past %d chunks", model, len(owing))
			}
			mu.Lock()
			res.Owing += len(owing)
			mu.Unlock()
			after = next

			chunks, texts, err := u.read(gctx, &source, owing, &res, &mu)
			if err != nil {
				return err
			}

			at := 0
			for _, batch := range embedding.Batches(texts, u.BatchCharacters) {
				prep, err := u.prepareBatch(gctx, model, chunks[at:at+len(batch)], batch)
				if err != nil {
					return err
				}
				at += len(batch)
				select {
				case <-gctx.Done():
					return gctx.Err()
				case preparedChan <- prep:
				}
			}
		}
	})

	g.Go(func() error {
		defer close(completedChan)
		for {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case prep, ok := <-preparedChan:
				if !ok {
					return nil
				}
				completed, err := u.inferBatch(gctx, prep)
				if err != nil {
					return err
				}
				select {
				case <-gctx.Done():
					return gctx.Err()
				case completedChan <- completed:
				}
			}
		}
	})

	g.Go(func() error {
		for {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case comp, ok := <-completedChan:
				if !ok {
					return nil
				}
				if len(comp.vectors) > 0 {
					if err := u.vectors.SaveVectors(gctx, comp.vectors); err != nil {
						return fmt.Errorf("write %d vectors: %w", len(comp.vectors), err)
					}
				}
				updateProgress(comp.reading, len(comp.vectors), comp.reused)
			}
		}
	})

	if err := g.Wait(); err != nil {
		return res, err
	}
	return res, nil
}

// read recovers what each chunk holds by opening its source again. The index
// keeps where a chunk is, not what it says.
//
// A chunk whose source is gone, or whose place is not in the text that source
// holds now, is left as it is: the file is what is true, and the index follows it
// when the file is next read.
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
		if p.Source != res.Reading {
			res.Reading = p.Source
			snapshot := *res
			mu.Unlock()
			u.progress(snapshot)
		} else {
			mu.Unlock()
		}
		prose, ok, err := source.textOf(ctx, p.Source, p.Producer, p.SourceHash)
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
	model port.EmbeddingModel,
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
	kept, err := u.vectors.GetKeptVectors(ctx, recipe, hashes)
	if err != nil {
		return preparedBatch{}, fmt.Errorf("what is already made: %w", err)
	}

	prep := preparedBatch{
		model:     model,
		owing:     owing,
		hashes:    hashes,
		asking:    make([]string, 0, len(texts)),
		askingMap: make([]int, len(texts)),
		reading:   owing[len(owing)-1].Source,
	}
	var seen map[string]int
	if len(texts) > 1 {
		seen = make(map[string]int, len(texts))
	}
	for i := range texts {
		value, held := kept[hex.EncodeToString(hashes[i])]
		if !held || len(hashes[i]) == 0 {
			key := owing[i].ChunkHash
			if key == "" {
				key = texts[i]
			}
			if seen != nil {
				if pos, exists := seen[key]; exists {
					prep.askingMap[i] = pos
					continue
				}
				seen[key] = len(prep.asking)
			}
			prep.askingMap[i] = len(prep.asking)
			prep.asking = append(prep.asking, texts[i])
			continue
		}
		prep.askingMap[i] = -1
		prep.reused = append(prep.reused, port.Vector{
			ChunkID: owing[i].ChunkID,
			Hash:    hashes[i],
			Model:   model,
			Kind:    port.QuantisedInt8,
			Value:   value,
			Coarse:  embedding.Coarse(embedding.Dimensions(value)),
		})
		prep.reusedCount++
	}
	return prep, nil
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
	out := make([]port.Vector, 0, len(prep.owing))
	out = append(out, prep.reused...)

	if len(prep.asking) > 0 {
		vectors, err := u.Embedder.Embed(ctx, prep.asking)
		if err != nil {
			return completedBatch{}, fmt.Errorf("embed %d chunks: %w", len(prep.asking), err)
		}
		if len(vectors) != len(prep.asking) {
			return completedBatch{}, fmt.Errorf("%s answered with %d vectors for %d texts", prep.model, len(vectors), len(prep.asking))
		}
		unique := make([]uniqueVector, len(vectors))
		for i, v := range vectors {
			if len(v) != prep.model.Dimensions {
				return completedBatch{}, fmt.Errorf("%s answered with %d dimensions", prep.model, len(v))
			}
			v = embedding.Normalise(v)
			quantised := embedding.Bytes(v)
			unique[i] = uniqueVector{
				value:  packVector(quantised),
				coarse: embedding.Coarse(quantised),
			}
		}
		for i, at := range prep.askingMap {
			if at >= 0 {
				out = append(out, port.Vector{
					ChunkID: prep.owing[i].ChunkID,
					Hash:    prep.hashes[i],
					Model:   prep.model,
					Kind:    port.QuantisedInt8,
					Value:   unique[at].value,
					Coarse:  unique[at].coarse,
				})
			}
		}
	}
	return completedBatch{
		vectors: out,
		reused:  prep.reusedCount,
		reading: prep.reading,
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

// extracted is the text a chunk's offsets belong to, taken out of the source
// again. The source now open is kept, because the chunks of one source are asked
// about together.
type extracted struct {
	of     text.Reader
	path   string
	text   string
	isHeld bool
}

// text is the extracted text of one source, and false where the vault no longer
// holds a source with text at that path.
//
// One reader for every kind and for every place a text may live, so that a
// chunk is re-sliced out of the text it was cut from.
func (e *extracted) textOf(ctx context.Context, path, from, hash string) (string, bool, error) {
	if e.path == path {
		return e.text, e.isHeld, nil
	}
	e.path, e.text, e.isHeld = path, "", false

	doc, err := e.of.GetDocument(ctx, path, from, hash)
	switch {
	case port.IsNoNote(err), errors.Is(err, fs.ErrNotExist), errors.Is(err, text.ErrUnreadable):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("read %s: %w", path, err)
	}
	e.text = doc.Text
	e.isHeld = true
	return e.text, true, nil
}
