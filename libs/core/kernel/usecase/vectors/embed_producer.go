package vectors

import (
	"cmp"
	"context"
	"encoding/hex"
	"fmt"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
)

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
	model       searchsource.Description
	place       searchsource.Place
	pending     []domain.Passage
	hashes      [][]byte
	reused      map[int]vector.Stored
	reusedCount int
	asking      []searchsource.Text
	askingMap   []int
	sources     []string
	id          uint64
}

// producer reads the chunks that lack vectors and cuts them into batches with the vectors already kept.
type producer struct {
	chunks          searchsourceport.VectorQueries
	vectors         searchsourceport.VectorRepository
	batchCharacters int
	isSkipped       func(path, hash string) bool
}

func newProducer(u Embed, isSkipped func(path, hash string) bool) producer {
	return producer{chunks: u.chunks, vectors: u.vectors, batchCharacters: u.BatchCharacters, isSkipped: isSkipped}
}

func (u producer) produceBatches(
	ctx context.Context,
	v domain.Vault,
	model searchsource.Description,
	source *extracted,
	track *tracker,
	out chan<- preparedBatch,
) error {
	res, mu := track.res, track.mu
	var after vector.ChunkCursor
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		pending, next, err := u.chunks.GetUnembeddedChunks(ctx, v.ID, model.Recipe(), after, chunksPerQuery)
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
		place := searchsource.Place{VaultID: v.ID, Recipe: model.Recipe()}
		if err := u.sendBatches(ctx, model, place, chunks, texts, track, out); err != nil {
			return err
		}
	}
}

func (u producer) sendBatches(
	ctx context.Context,
	model searchsource.Description,
	place searchsource.Place,
	chunks []domain.Passage,
	texts []string,
	track *tracker,
	out chan<- preparedBatch,
) error {
	pos := 0
	for _, batch := range splitBatches(texts, u.batchCharacters) {
		prep, err := u.prepareBatch(ctx, model, place, chunks[pos:pos+len(batch)], batch)
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
	model searchsource.Description,
	place searchsource.Place,
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
		place:     place,
		pending:   pending,
		hashes:    hashes,
		reused:    make(map[int]vector.Stored),
		asking:    make([]searchsource.Text, 0, len(texts)),
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
			prep.queueText(i, key, searchsource.Text{ChunkID: pending[i].ChunkID, Path: pending[i].Source, VersionID: pending[i].Producer, Text: texts[i]}, deduplicated)
			continue
		}
		prep.askingMap[i] = -1
		prep.reused[i] = vector.Stored{
			ChunkID: pending[i].ChunkID,
			Hash:    hashes[i],
			Recipe:  recipe,
			Kind:    vector.QuantizedInt8,
			Value:   value,
			Coarse:  vector.Coarse(vector.Dimensions(value)),
		}
		prep.reusedCount++
	}
	return prep, nil
}

func (p *preparedBatch) queueText(i int, key string, value searchsource.Text, visited map[string]int) {
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
