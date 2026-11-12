package vectors

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// completedBatch is one batch whose vectors are ready to be stored.
type completedBatch struct {
	vectors []embedding.Vector
	reused  int
	id      uint64
}

// inferrer embeds the texts of the batches the producer cut.
type inferrer struct {
	embedder embeddingport.Embedder
}

// storer writes the vectors of the completed batches.
type storer struct {
	vectors embeddingport.VectorRepository
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
