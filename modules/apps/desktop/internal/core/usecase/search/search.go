package search

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/text"
)

// defaultLimit is how many results a caller that names no number gets.
const defaultLimit = 20

// lexicalCandidates is how many candidates the words keep for every result
// the search returns, so the answer is among what it ranks.
const lexicalCandidates = 5

// DefaultFloor is how near the query a passage stands to be an answer, in
// cosine similarity.
//
// A nearest-neighbour index answers with as many rows as it is asked for
// whatever the question, and this is what leaves a vault holding nothing near
// with nothing to say. The number belongs to the model the vectors were made
// by: it is where that model puts two pieces of text about different things,
// and another model puts them somewhere else.
const DefaultFloor = 0.50

// Parameters says which ways a search is asked, how many candidates each keeps,
// and what a candidate has to reach. A half that keeps none does not run; with
// neither named, both run.
type Parameters struct {
	// Limit is how many results come back.
	Limit int
	// Each is how many passages one document may answer with. Zero is one, and
	// a document names itself once.
	//
	// A reader who cannot turn the page needs the several places a book speaks
	// about a thing, and a list a person runs their eye down needs one line per
	// book.
	Each int
	// Lexical is how many candidates the words keep, and Dense how many passages
	// the meaning returns. The coarse pass under it keeps several
	// times as many, and the full-precision vectors order those.
	Lexical int
	Dense   int
	// Named is how many sections the names keep. A document has far fewer
	// sections than chunks, and one name matching is a strong thing to have
	// said, so a few of them are enough.
	Named int
	// Floor is how near the query a passage stands to be an answer at all. It
	// is read only by the meaning half; the words half has no distance.
	Floor float64
	// Growing says the last word typed may still be being typed, so the index
	// matches it by its opening. A question that is finished is asked exactly.
	Growing bool
	// Of are the kinds of source the question is about. None is every kind,
	// which is what a question that says nothing about the sort of file it
	// wants asks for.
	//
	// A person asking a book about something is asking about the book. Told to
	// look everywhere, an answer draws whatever the vault holds most of.
	Of []domain.SourceKind
}

// filled supplies what the caller left out.
func (p Parameters) filled() Parameters {
	if p.Limit <= 0 {
		p.Limit = defaultLimit
	}
	if p.Each <= 0 {
		p.Each = 1
	}
	if p.Lexical <= 0 && p.Dense <= 0 && p.Named <= 0 {
		p.Lexical = p.Limit * lexicalCandidates
		p.Dense = p.Limit
		p.Named = p.Limit
	}
	if p.Floor == 0 {
		p.Floor = DefaultFloor
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
	derived  port.DerivedStores
	floor    float64
}

// New is a search over one vault's index.
//
// The embedder turns a query into a vector. Nil is an installation with no
// model, and then the meaning half does not run and the words half answers alone,
// which is a whole search: the vector index is optional and may never finish
// filling.
//
// `floor` is how near the query a passage stands to be an answer, in the units
// the model in use measures in. Zero takes DefaultFloor.
func New(passages port.PassageQueries, readers port.VaultReaders, derived port.DerivedStores, embedder port.Embedder, floor float64) Search {
	if floor == 0 {
		floor = DefaultFloor
	}
	return Search{
		passages: passages,
		readers:  readers,
		embedder: embedder,
		derived:  derived,
		floor:    floor,
	}
}

// Execute runs the ways the parameters name, merges their rankings by rank,
// and returns one passage per document.
//
// Three orders: the words in a chunk, the meaning of a chunk, and the name of
// the section a chunk opens. A chunk that two of them place well outranks one
// that any of them placed first alone.
func (u Search) Execute(ctx context.Context, v domain.Vault, query string, p Parameters) ([]domain.Passage, error) {
	if p.Floor == 0 {
		p.Floor = u.floor
	}
	p = p.filled()

	var rankings [][]domain.Passage
	if p.Lexical > 0 {
		lexical, err := u.passages.Lexical(ctx, v.ID, query, p.Of, p.Lexical, p.Growing)
		if err != nil {
			return nil, err
		}
		rankings = append(rankings, lexical)
	}
	var named []domain.Passage
	if p.Named > 0 {
		found, err := u.passages.Named(ctx, v.ID, query, p.Of, p.Named, p.Growing)
		if err != nil {
			return nil, err
		}
		named = found
		rankings = append(rankings, named)
	}
	if p.Dense > 0 && u.embedder != nil {
		dense, err := u.nearest(ctx, v, query, p)
		if err != nil {
			return nil, err
		}
		rankings = append(rankings, dense)
	}
	return u.read(ctx, v, collapse(merge(rankings...), named, p.Each, p.Limit))
}

// nearest is the search asked by meaning, over a vector of the query itself.
func (u Search) nearest(ctx context.Context, v domain.Vault, query string, p Parameters) ([]domain.Passage, error) {
	vectors, err := u.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vectors) != 1 {
		return nil, fmt.Errorf("the embedder answered with %d vectors for one query", len(vectors))
	}
	// A vector is kept under the recipe it was made by, which is everything
	// about the model that decides what a vector is. Asked under anything else,
	// no vector is found and this half answers nothing at all.
	return u.passages.Nearest(ctx, v.ID, u.embedder.Model().Recipe(), vectors[0], p.Of, p.Dense, p.Floor)
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
	var store port.DerivedStore
	if u.derived != nil {
		if store, err = u.derived.Open(v); err != nil {
			return nil, err
		}
	}
	of := text.Reader{Vault: reader, Derived: store}

	// Several passages of one file are read once. A source is one text however
	// many passages name it, so its path is the whole of the key.
	read := map[string]string{}
	gone := map[string]bool{}

	out := make([]domain.Passage, 0, len(found))
	for _, p := range found {
		prose, held := read[p.Source]
		if !held && !gone[p.Source] {
			prose, err = extracted(ctx, of, p.Source, p.TextFrom, p.Hash)
			if port.NoNote(err) || errors.Is(err, errUnreadable) {
				gone[p.Source] = true
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", p.Source, err)
			}
			read[p.Source] = prose
		}
		if gone[p.Source] {
			continue
		}
		p.Text = span(prose, p.Start, p.Length)
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
//
// Which reader produces it is decided in one place, so that what a search slices
// and what an extractor cut are the same text.
func extracted(ctx context.Context, of text.Reader, path, from, hash string) (string, error) {
	doc, err := of.Of(ctx, path, from, hash)
	if errors.Is(err, text.ErrUnreadable) {
		return "", errUnreadable
	}
	if err != nil {
		return "", err
	}
	return doc.Text, nil
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

// Way is how a search is asked. Each way is an order of its own, and a search
// asked every way fuses them into one.
type Way int

const (
	// EveryWay: all of them, fused into one ranking.
	EveryWay Way = iota
	// ByWords: what is written, matched as words.
	ByWords
	// ByMeaning: what the query means, against the vectors the index holds.
	ByMeaning
	// ByName: the names of the sections a source divides into.
	ByName
)

// Typing is the parameters for a search asked the way named, while a person is
// still typing it: the last word is matched by its opening.
//
// A way that is not wanted keeps no candidates, which is how a way is told not
// to run. Every way but the one named is silenced, so a caller drawing the ways
// apart is shown one of them and not one and a half.
func Typing(way Way, limit int) Parameters {
	p := Parameters{Limit: limit, Growing: true}.filled()
	switch way {
	case ByWords:
		p.Dense, p.Named = 0, 0
	case ByMeaning:
		p.Lexical, p.Named = 0, 0
	case ByName:
		p.Lexical, p.Dense = 0, 0
	case EveryWay:
	}
	return p
}
