package source

import (
	"context"
	"errors"
	"io/fs"
	"sort"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/correction"
	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Highlight is what runs of a source's text say and where they sit on the pages
// they were read from.
//
// Two things place a word — a model reading a scan, and the document's own text
// layer — and which of them made the text the chunks are places in is the one
// question asked here. It is the question extraction asks when it cuts a source,
// answered by the same column: an offset belongs to one producer's text, and the
// other producer's rectangles cover other words.
//
// A source with no reading and no layer is lit nowhere, and that is an answer.
type Highlight struct {
	Readers port.VaultReaders
	Sources port.SourceQueries
	Derived port.DerivedStores

	// Documents is optional. It reads a document that carries its own text
	// layer; without one such a document is lit nowhere. A vault whose
	// documents are all recognised needs none.
	Documents port.TextExtractor
}

// A Run is one run of a source's text: what it says, and where it stands on the
// pages it was read from.
type Run struct {
	Text  string
	Boxes []highlight.Box
}

// NewHighlight is what a run of text says and where it stands on the pages it
// was read from: the vault the document is read out of, what says which producer
// made the text the runs are places in, and the store that producer's text and
// coordinates are kept in.
func NewHighlight(
	readers port.VaultReaders, sources port.SourceQueries, derived port.DerivedStores,
) Highlight {
	return Highlight{Readers: readers, Sources: sources, Derived: derived}
}

// Execute is what the runs of one source's text say and where each of them
// sits: the boxes covering it, each on the page it was read from.
//
// The answer stands in the order the runs were asked about, so a caller that
// asked about a passage and the places around it knows which is which. The text
// and the coordinates are read once however many runs are asked about.
func (u Highlight) Execute(
	ctx context.Context,
	v domain.Vault,
	path string,
	runs []domain.Span,
) ([]Run, error) {
	reader, err := u.Readers.Open(v)
	if err != nil {
		return nil, err
	}
	// Everything from outside reaches the vault through a reader, so a path
	// leaving it is refused there, and so is one the vault holds nothing at.
	ref, err := reader.Stat(ctx, path)
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, nil
	}
	said, stands, err := u.Sources.Reading(ctx, v.ID, path)
	if err != nil {
		return nil, err
	}
	// Which producer made a source's text is what says where its offsets are,
	// and a source the index does not hold says nothing about either.
	if !stands {
		return over("", nil, runs), nil
	}
	var store port.DerivedStore
	if u.Derived != nil {
		if store, err = u.Derived.Open(v); err != nil {
			return nil, err
		}
	}

	// A reading is of the bytes the index last saw, and its text and
	// coordinates describe those. A file rewritten since is read from its own
	// layer, which is the words that are there now.
	var boxes []highlight.Box
	if said.Producer != "" && ref.Unchanged(said.Fingerprint) {
		boxes, err = u.read(ctx, store, said)
	} else {
		said = port.SourceText{}
		boxes, err = u.layer(ctx, reader, ref, runs)
	}
	if err != nil {
		return nil, err
	}
	prose, err := u.prose(ctx, reader, store, path, said)
	if err != nil {
		return nil, err
	}
	return over(prose, boxes, runs), nil
}

// prose is the text the runs are places in: what the producer wrote, or the
// file's own words where no reading of these bytes stands. A source nothing
// here reads says nothing, and that is an answer.
func (u Highlight) prose(
	ctx context.Context,
	reader port.VaultReader,
	store port.DerivedStore,
	path string,
	said port.SourceText,
) (string, error) {
	of := text.Reader{Vault: reader, Derived: store, Documents: u.Documents}
	doc, err := of.Of(ctx, path, said.Producer, said.Hash)
	if errors.Is(err, text.ErrUnreadable) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return doc.Text, nil
}

// over is what each run says and where it sits, in the order the runs were
// asked about. A run standing nowhere is lit nowhere and keeps its place in the
// answer.
func over(prose string, boxes []highlight.Box, runs []domain.Span) []Run {
	out := make([]Run, 0, len(runs))
	for _, one := range runs {
		start, length := getRuneBounds(prose, one.From, one.Len())
		out = append(out, Run{
			Text:  prose[start : start+length],
			Boxes: highlight.Over(boxes, one),
		})
	}
	return out
}

// read is where a producer put the words it read. The coordinates are kept
// beside the text they were read with, under the hash of the bytes both came
// from.
func (u Highlight) read(
	ctx context.Context,
	store port.DerivedStore,
	said port.SourceText,
) ([]highlight.Box, error) {
	if store == nil {
		return nil, nil
	}
	raw, err := store.Read(ctx, text.Boxes(said.Producer, said.Hash))
	if errors.Is(err, fs.ErrNotExist) {
		// The store is a folder on the person's disk and they may empty it.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	boxes := highlight.Unpack(raw)

	// A reading that was proofread is read with its corrections in it, so a run
	// of its text is a run of the corrected prose and the coordinates say where
	// those words stand.
	corrections, err := store.Read(ctx, text.Corrections(said.Producer, said.Hash))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return correction.Boxes(boxes, correction.Unpack(corrections)), nil
}

// layer is where the document's own text layer put the words, over the pages
// the runs fall on and no others.
//
// Which pages those are comes from where each page's text begins, which the
// document says when it is read. A page nothing asked about is not read.
func (u Highlight) layer(
	ctx context.Context,
	reader port.VaultReader,
	ref domain.Fingerprint,
	runs []domain.Span,
) ([]highlight.Box, error) {
	if name, ok := text.ReaderName(ref); !ok || name != text.ReaderPDF {
		// A book made for a screen is set afresh wherever it is shown, and
		// carries no rectangles.
		return nil, nil
	}
	if u.Documents == nil {
		return nil, nil
	}
	raw, err := reader.Read(ctx, ref.Path)
	if err != nil {
		return nil, err
	}
	book, err := u.Documents.Read(ctx, raw)
	if err != nil {
		return nil, err
	}
	pages := every(book, runs)
	if len(pages) == 0 {
		return nil, nil
	}
	return u.Documents.Highlights(ctx, raw, book.Pages, pages)
}

// every is the pages all the runs fall on, in order and each of them once. Two
// runs on one page are one page read.
func every(book port.TextLayer, runs []domain.Span) []int {
	held := map[int]bool{}
	var out []int
	for _, one := range runs {
		for _, page := range across(book, one.From, one.To) {
			if held[page] {
				continue
			}
			held[page] = true
			out = append(out, page)
		}
	}
	sort.Ints(out)
	return out
}

// across is the pages a run of the document's text falls on. A page holds the
// text from where it begins up to where the next page does, and the last page
// holds the rest.
func across(book port.TextLayer, start, end int) []int {
	if end > len(book.Text) {
		end = len(book.Text)
	}
	if start >= end || len(book.Pages) == 0 {
		return nil
	}
	// The page the run begins on is the last one beginning at or before it.
	first := sort.Search(len(book.Pages), func(i int) bool {
		return book.Pages[i] > start
	}) - 1
	if first < 0 {
		first = 0
	}
	var out []int
	for i := first; i < len(book.Pages) && book.Pages[i] < end; i++ {
		out = append(out, i)
	}
	return out
}
