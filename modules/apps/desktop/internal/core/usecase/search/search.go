package search

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

// defaultLimit is how many results a caller that names no number gets.
const defaultLimit = 20

// candidatesPerResult is how many candidates a half keeps for each result the
// search returns, so the answer is among its candidates.
const candidatesPerResult = 5

// Parameters says which halves of a search run and how many candidates each
// keeps. A half that keeps none does not run; with neither named, both run.
type Parameters struct {
	// Limit is how many results come back, one per document.
	Limit int
	// Lexical is how many candidates the words half keeps.
	Lexical int
	// Dense is how many candidates the coarse pass keeps.
	Dense int
}

// filled supplies what the caller left out.
func (p Parameters) filled() Parameters {
	if p.Limit <= 0 {
		p.Limit = defaultLimit
	}
	if p.Lexical <= 0 && p.Dense <= 0 {
		p.Lexical = p.Limit * candidatesPerResult
		p.Dense = p.Limit * candidatesPerResult
	}
	return p
}

// Search answers a query within one vault, over everything the vault holds.
//
// One database holds every vault, so the vault is an argument of the search.
// Every field is asked for by New, since none of them is a parameter a caller
// may forget. An embedder that is left out answers: the words half runs alone,
// quickly, and with none of what the vectors hold. Absence is said by passing
// nothing.
type Search struct {
	passages port.PassageQueries
	readers  port.VaultReaders
	embedder port.Embedder
}

// New is a search over one vault's index.
//
// The embedder turns a query into a vector. Nil is an installation with no
// model, and then the coarse pass does not run and the words half answers alone,
// which is a whole search: the vector index is optional and may never finish
// filling.
func New(passages port.PassageQueries, readers port.VaultReaders, embedder port.Embedder) Search {
	return Search{passages: passages, readers: readers, embedder: embedder}
}

// Execute runs the halves the parameters name, merges their rankings by rank,
// and returns one passage per document.
func (u Search) Execute(ctx context.Context, v domain.Vault, query string, p Parameters) ([]domain.Passage, error) {
	p = p.filled()

	var rankings [][]domain.Passage
	if p.Lexical > 0 {
		lexical, err := u.passages.Lexical(ctx, v.ID, query, p.Lexical)
		if err != nil {
			return nil, err
		}
		rankings = append(rankings, lexical)
	}
	if p.Dense > 0 && u.embedder != nil {
		dense, err := u.nearest(ctx, v, query, p.Dense)
		if err != nil {
			return nil, err
		}
		rankings = append(rankings, dense)
	}
	return u.read(ctx, v, collapse(merge(rankings...), p.Limit))
}

// nearest is the coarse pass, over a vector of the query itself.
func (u Search) nearest(ctx context.Context, v domain.Vault, query string, k int) ([]domain.Passage, error) {
	vectors, err := u.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vectors) != 1 {
		return nil, fmt.Errorf("the embedder answered with %d vectors for one query", len(vectors))
	}
	return u.passages.Nearest(ctx, v.ID, embedding.Bits(vectors[0]), k)
}

// read fills in the text of each passage from the vault. A chunk is a place in a
// file, and the file is what holds the words.
//
// A passage naming a file the vault no longer holds is dropped: the file is what
// is true, and the index follows it.
func (u Search) read(ctx context.Context, v domain.Vault, found []domain.Passage) ([]domain.Passage, error) {
	if len(found) == 0 {
		return nil, nil
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return nil, err
	}

	// Several passages of one file are read once.
	read := map[string]string{}
	gone := map[string]bool{}

	out := make([]domain.Passage, 0, len(found))
	for _, p := range found {
		text, held := read[p.Source]
		if !held && !gone[p.Source] {
			text, err = extracted(ctx, reader, p.Source)
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, errUnreadable) {
				gone[p.Source] = true
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", p.Source, err)
			}
			read[p.Source] = text
		}
		if gone[p.Source] {
			continue
		}
		p.Text = span(text, p.Start, p.Length)
		out = append(out, p)
	}
	return out, nil
}

// errUnreadable is a source that is there and says nothing this can use. It is
// not a failure of the search: the passage is dropped and the rest answer.
var errUnreadable = errors.New("nothing could be read from the source")

// extracted is the text a source's chunks are places in.
//
// A note's text is its file. A book's is what taking the text out of it produces,
// and a chunk's offsets belong to that, not to the bytes on disk: slicing the
// archive at a text offset returns compressed noise. Extraction is deterministic,
// which is what lets the text be recovered from the source it belongs to.
func extracted(ctx context.Context, reader port.VaultReader, path string) (string, error) {
	ref, err := reader.Stat(ctx, path)
	if err != nil {
		return "", err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return "", err
	}
	if ref.Kind == domain.KindBook {
		book, err := epub.Read(raw)
		if err != nil {
			return "", errUnreadable
		}
		return book.Text, nil
	}
	return string(raw), nil
}

// span is the text a window addresses, bounded by what the source holds now. A
// source edited since it was indexed is shorter than the window says.
func span(raw string, start, length int) string {
	if start < 0 || start >= len(raw) || length <= 0 {
		return ""
	}
	end := start + length
	if end > len(raw) {
		end = len(raw)
	}
	return raw[start:end]
}
