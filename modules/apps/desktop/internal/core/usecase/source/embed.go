package source

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/embedding"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/epub"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
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
	Readers port.VaultReaders
	Chunks  port.VectorQueries
	Vectors port.VectorRepository

	// Embedder is optional. Without one nothing is embedded and a search answers
	// on its words alone, which is a whole search: the vector index fills in
	// behind, and may never finish.
	Embedder port.Embedder

	// BatchCharacters bounds one request to the model. Zero takes the default.
	BatchCharacters int

	// OnProgress, if set, is called each time a group of vectors is written.
	OnProgress func(EmbedResult)
}

// EmbedResult reports what embedding did.
type EmbedResult struct {
	Owing     int    // chunks found with no vector from the model in use
	Embedded  int    // chunks that now carry one
	Vanished  int    // whose source the vault no longer holds
	Displaced int    // whose place is not in the text their source holds now
	Reading   string // the source open now
}

// Execute embeds what one vault owes, in groups, until nothing owes anything.
func (u Embed) Execute(ctx context.Context, v domain.Vault) (EmbedResult, error) {
	var res EmbedResult
	if u.Embedder == nil {
		return res, nil
	}

	model := u.Embedder.Model()
	reader, err := u.Readers.Open(v)
	if err != nil {
		return res, err
	}
	source := extracted{reader: reader}

	after := int64(0)
	for {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		owing, err := u.Chunks.Unembedded(ctx, v.ID, model, after, chunksPerQuery)
		if err != nil {
			return res, fmt.Errorf("what owes a vector from %s: %w", model, err)
		}
		if len(owing) == 0 {
			return res, nil
		}
		res.Owing += len(owing)
		after = owing[len(owing)-1].Chunk

		chunks, texts, err := u.read(ctx, &source, owing, &res)
		if err != nil {
			return res, err
		}

		at := 0
		for _, batch := range embedding.Batches(texts, u.BatchCharacters) {
			if err := u.write(ctx, model, chunks[at:at+len(batch)], batch, &res); err != nil {
				return res, err
			}
			at += len(batch)
		}
	}
}

// read recovers what each chunk holds by opening its source again. The index
// keeps where a chunk is, not what it says.
//
// A chunk whose source is gone, or whose place is not in the text that source
// holds now, is left as it is: the file is what is true, and the index follows it
// when the file is next read.
func (u Embed) read(ctx context.Context, source *extracted, owing []domain.Passage, res *EmbedResult) ([]int64, []string, error) {
	chunks := make([]int64, 0, len(owing))
	texts := make([]string, 0, len(owing))

	for _, p := range owing {
		if p.Source != res.Reading {
			res.Reading = p.Source
			u.progress(*res)
		}
		text, ok, err := source.of(ctx, p.Source)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			res.Vanished++
			continue
		}
		if p.Start < 0 || p.Length <= 0 || p.Start+p.Length > len(text) {
			res.Displaced++
			continue
		}
		chunks = append(chunks, p.Chunk)
		texts = append(texts, text[p.Start:p.Start+p.Length])
	}
	return chunks, texts, nil
}

// write embeds one group and stores both representations of every vector in it.
//
// The group is written before the next one is asked for, so everything a run
// embedded is stored, and both representations of a vector are one value: a chunk
// holding one and not the other is absent from the coarse pass and invisible to
// the question of what has no vector.
func (u Embed) write(ctx context.Context, model port.EmbeddingModel, chunks []int64, texts []string, res *EmbedResult) error {
	if len(texts) == 0 {
		return nil
	}
	vectors, err := u.Embedder.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("embed %d chunks: %w", len(texts), err)
	}
	if len(vectors) != len(texts) {
		return fmt.Errorf("%s answered with %d vectors for %d texts", model, len(vectors), len(texts))
	}

	out := make([]port.Vector, 0, len(vectors))
	for i, v := range vectors {
		if len(v) != model.Dimensions {
			return fmt.Errorf("%s answered with %d dimensions", model, len(v))
		}
		// The scale that turns a float into a byte belongs to the model and is
		// fixed, and it is fixed for a vector of unit length.
		v = embedding.Normalise(v)
		out = append(out, port.Vector{
			Chunk:  chunks[i],
			Model:  model,
			Kind:   port.QuantisedInt8,
			Value:  signed(embedding.Bytes(v)),
			Coarse: embedding.Bits(v),
		})
	}
	if err := u.Vectors.SaveVectors(ctx, out); err != nil {
		return fmt.Errorf("write %d vectors: %w", len(out), err)
	}
	res.Embedded += len(out)
	u.progress(*res)
	return nil
}

func (u Embed) progress(res EmbedResult) {
	if u.OnProgress != nil {
		u.OnProgress(res)
	}
}

// signed is a quantised vector as the bytes that are stored, one per dimension.
func signed(q []int8) []byte {
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
	reader port.VaultReader
	path   string
	text   string
	held   bool
}

// of is the extracted text of one source, and false where the vault no longer
// holds a source with text at that path.
func (e *extracted) of(ctx context.Context, path string) (string, bool, error) {
	if e.path == path {
		return e.text, e.held, nil
	}
	e.path, e.text, e.held = path, "", false

	ref, err := e.reader.Stat(ctx, path)
	if port.NoNote(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("stat %s: %w", path, err)
	}
	raw, err := e.reader.Read(ctx, path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read %s: %w", path, err)
	}

	switch ref.Kind {
	case domain.KindBook:
		book, err := epub.Read(raw)
		if err != nil {
			return "", false, nil
		}
		e.text = book.Text
	default:
		// A note's text is the file, and its chunks are places in it.
		e.text = string(raw)
	}
	e.held = true
	return e.text, true, nil
}
