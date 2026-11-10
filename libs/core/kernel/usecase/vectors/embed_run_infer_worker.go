package vectors

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	indexdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/index"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vector"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

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

func (u Embed) storeBatches(
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

func (u Embed) saveCompleted(ctx context.Context, comp completedBatch) error {
	if len(comp.vectors) == 0 {
		return nil
	}
	if err := u.vectors.SaveVectors(ctx, comp.vectors); err != nil {
		return fmt.Errorf("write %d vectors: %w", len(comp.vectors), err)
	}
	return nil
}

func (u Embed) read(
	ctx context.Context,
	source *extracted,
	pending []domain.Passage,
	res *indexdomain.EmbedResult,
	mu *sync.Mutex,
	track *tracker,
) ([]domain.Passage, []string, error) {
	chunks := make([]domain.Passage, 0, len(pending))
	texts := make([]string, 0, len(pending))

	isFirst := true
	for _, p := range pending {
		if u.IsSkipped != nil && u.IsSkipped(p.Source, p.ContentHash) {
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

func (u Embed) prepareBatch(
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
			Kind:    embedding.QuantisedInt8,
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

func (u Embed) inferBatch(ctx context.Context, prep preparedBatch) (completedBatch, error) {
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
				Kind:    embedding.QuantisedInt8,
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

func (u Embed) progress(res indexdomain.EmbedResult) {
	if u.OnProgress != nil {
		u.OnProgress(res)
	}
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
	case edit.IsNoNote(err), errors.Is(err, edit.ErrNotExist), errors.Is(err, textdomain.ErrNoText):
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

func (u Embed) embedAsking(ctx context.Context, prep preparedBatch) ([]uniqueVector, error) {
	if len(prep.asking) == 0 {
		return nil, nil
	}
	vectors, err := u.Embedder.Embed(ctx, prep.asking)
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
		quantised := vector.Bytes(vector.Normalise(v))
		unique[i] = uniqueVector{
			value:  packVector(quantised),
			coarse: vector.Coarse(quantised),
		}
	}
	return unique, nil
}
