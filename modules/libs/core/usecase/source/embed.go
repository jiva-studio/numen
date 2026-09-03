package source

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
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

	// Derived holds what a recogniser wrote. A chunk of a recognised document
	// is re-sliced out of that and not out of the document.
	Derived port.DerivedStore
	// Documents reads a format that needs a library, for a source standing on
	// its own bytes.
	Documents port.Documents

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
	Owing    int // chunks found with no vector from the model in use
	Embedded int // chunks that now carry one
	// Reused counts the chunks whose vector was already made for their text and
	// was claimed rather than bought.
	Reused    int
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
	var store port.DerivedStore
	if u.Derived != nil {
		store = u.Derived
	}
	source := extracted{of: text.Reader{Vault: reader, Derived: store, Documents: u.Documents}}

	after := int64(0)
	for {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		owing, err := u.Chunks.Unembedded(ctx, string(v.ID), model, after, chunksPerQuery)
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
func (u Embed) read(ctx context.Context, source *extracted, owing []domain.Passage, res *EmbedResult) ([]domain.Passage, []string, error) {
	chunks := make([]domain.Passage, 0, len(owing))
	texts := make([]string, 0, len(owing))

	for _, p := range owing {
		if p.Source != res.Reading {
			res.Reading = p.Source
			u.progress(*res)
		}
		prose, ok, err := source.textOf(ctx, p.Source, p.TextFrom, p.Hash)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			res.Vanished++
			continue
		}
		if p.Start < 0 || p.Length <= 0 || p.Start+p.Length > len(prose) {
			res.Displaced++
			continue
		}
		chunks = append(chunks, p)
		texts = append(texts, prose[p.Start:p.Start+p.Length])
	}
	return chunks, texts, nil
}

// write embeds one group and stores both representations of every vector in it.
//
// The group is written before the next one is asked for, so everything a run
// embedded is stored, and both representations of a vector are one value: a chunk
// holding one and not the other is absent from the coarse pass and invisible to
// the question of what has no vector.
func (u Embed) write(ctx context.Context, model port.EmbeddingModel, owing []domain.Passage, texts []string, res *EmbedResult) error {
	if len(texts) == 0 {
		return nil
	}
	recipe := model.Recipe()

	// The text each chunk holds, as the index recorded it when the source was
	// cut. That record is what a chunk is identified by, and what every
	// question about what still owes a vector is asked against.
	prints := make([][]byte, len(texts))
	for i := range texts {
		raw, err := hex.DecodeString(owing[i].Fingerprint)
		if err != nil || len(raw) == 0 {
			continue
		}
		prints[i] = raw
	}
	kept, err := u.Vectors.Kept(ctx, recipe, prints)
	if err != nil {
		return fmt.Errorf("what is already made: %w", err)
	}

	out := make([]port.Vector, 0, len(texts))
	var asking []string
	var askingFor []int
	for i := range texts {
		value, held := kept[hex.EncodeToString(prints[i])]
		if !held || len(prints[i]) == 0 {
			asking = append(asking, texts[i])
			askingFor = append(askingFor, i)
			continue
		}
		// Bought once. What the coarse pass needs is read back out of it.
		out = append(out, port.Vector{
			Chunk:       owing[i].Chunk,
			Fingerprint: prints[i],
			Model:       model,
			Kind:        port.QuantisedInt8,
			Value:       value,
			Coarse:      embedding.Coarse(unsigned(value)),
		})
		res.Reused++
	}

	if len(asking) > 0 {
		vectors, err := u.Embedder.Embed(ctx, asking)
		if err != nil {
			return fmt.Errorf("embed %d chunks: %w", len(asking), err)
		}
		if len(vectors) != len(asking) {
			return fmt.Errorf("%s answered with %d vectors for %d texts", model, len(vectors), len(asking))
		}
		for i, v := range vectors {
			if len(v) != model.Dimensions {
				return fmt.Errorf("%s answered with %d dimensions", model, len(v))
			}
			// The scale that turns a float into a byte belongs to the model and
			// is fixed, and it is fixed for a vector of unit length.
			v = embedding.Normalise(v)
			at := askingFor[i]
			quantised := embedding.Bytes(v)
			out = append(out, port.Vector{
				Chunk:       owing[at].Chunk,
				Fingerprint: prints[at],
				Model:       model,
				Kind:        port.QuantisedInt8,
				Value:       signed(quantised),
				Coarse:      embedding.Coarse(quantised),
			})
		}
	}

	if err := u.Vectors.SaveVectors(ctx, out); err != nil {
		return fmt.Errorf("write %d vectors: %w", len(out), err)
	}
	res.Embedded += len(out)
	u.progress(*res)
	return nil
}

// unsigned is a stored vector as the dimensions it holds, one per byte.
func unsigned(stored []byte) []int8 {
	out := make([]int8, len(stored))
	for i, b := range stored {
		out[i] = int8(b)
	}
	return out
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
	of   text.Reader
	path string
	text string
	held bool
}

// text is the extracted text of one source, and false where the vault no longer
// holds a source with text at that path.
//
// One reader for every kind and for every place a text may live, so that a
// chunk is re-sliced out of the text it was cut from.
func (e *extracted) textOf(ctx context.Context, path, from, hash string) (string, bool, error) {
	if e.path == path {
		return e.text, e.held, nil
	}
	e.path, e.text, e.held = path, "", false

	doc, err := e.of.Of(ctx, path, from, hash)
	switch {
	case port.NoNote(err), errors.Is(err, fs.ErrNotExist), errors.Is(err, text.ErrUnreadable):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("read %s: %w", path, err)
	}
	e.text = doc.Text
	e.held = true
	return e.text, true, nil
}
