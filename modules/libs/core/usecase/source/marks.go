package source

import (
	"context"
	"errors"
	"io/fs"
	"sort"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/fixes"
	"github.com/jiva-studio/numen/modules/libs/core/lit"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

// Marks says where a run of a source's text sits on the pages it was read from.
//
// Two things place a word — a model reading a scan, and the document's own text
// layer — and which of them made the text the chunks are places in is the one
// question asked here. It is the question extraction asks when it cuts a source,
// answered by the same column: an offset belongs to one producer's text, and the
// other producer's rectangles cover other words.
//
// A source with no reading and no layer is lit nowhere, and that is an answer.
type Marks struct {
	Readers port.VaultReaders
	Sources port.SourceQueries
	Derived port.DerivedStores

	// Documents reads a document that carries its own text layer. A vault whose
	// documents are all recognised needs none.
	Documents port.Documents
}

// Execute is where the runs of one source's text sit: for each of them, the
// pages it falls on and, on each, the rectangles covering it.
//
// The answer stands in the order the runs were asked about, so a caller that
// asked about a passage and the places around it knows which is which. The
// coordinates are read once however many runs are asked about.
func (u Marks) Execute(
	ctx context.Context,
	v domain.Vault,
	path string,
	runs []lit.Run,
) ([][]lit.Page, error) {
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

	said, held, err := u.Sources.Reading(ctx, v.ID, path)
	if err != nil || !held {
		return nil, err
	}
	// A reading is of the bytes the index last saw, and its coordinates
	// describe those. A file rewritten since is read from its own layer, which
	// is the words that are there now.
	var boxes []lit.Box
	if said.From != "" && ref.Unchanged(domain.FileRef{Size: said.Size, MTime: said.MTime}) {
		boxes, err = u.read(ctx, v, said)
	} else {
		boxes, err = u.layer(ctx, reader, ref, runs)
	}
	if err != nil {
		return nil, err
	}
	return over(boxes, runs), nil
}

// over is where each run sits, in the order the runs were asked about. A run
// standing nowhere is lit nowhere and keeps its place in the answer.
func over(boxes []lit.Box, runs []lit.Run) [][]lit.Page {
	out := make([][]lit.Page, 0, len(runs))
	for _, one := range runs {
		out = append(out, lit.Marks(boxes, one.Start, one.Length))
	}
	return out
}

// read is where a producer put the words it read. The coordinates are kept
// beside the text they were read with, under the hash of the bytes both came
// from.
func (u Marks) read(
	ctx context.Context,
	v domain.Vault,
	said port.Recognised,
) ([]lit.Box, error) {
	if u.Derived == nil {
		return nil, nil
	}
	store, err := u.Derived.Open(v)
	if err != nil {
		return nil, err
	}
	raw, err := store.Read(ctx, text.Boxes(said.From, said.Hash))
	if errors.Is(err, fs.ErrNotExist) {
		// The store is a folder on the person's disk and they may empty it.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	boxes := lit.Unpack(raw)

	// A reading that was proofread is read with its corrections in it, so a run
	// of its text is a run of the corrected prose and the coordinates say where
	// those words stand.
	corrections, err := store.Read(ctx, text.Fixes(said.From, said.Hash))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return fixes.Boxes(boxes, fixes.Unpack(corrections)), nil
}

// layer is where the document's own text layer put the words, over the pages
// the runs fall on and no others.
//
// Which pages those are comes from where each page's text begins, which the
// document says when it is read. A page nothing asked about is not read.
func (u Marks) layer(
	ctx context.Context,
	reader port.VaultReader,
	ref domain.FileRef,
	runs []lit.Run,
) ([]lit.Box, error) {
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
	return u.Documents.Lit(ctx, raw, book.Pages, pages)
}

// every is the pages all the runs fall on, in order and each of them once. Two
// runs on one page are one page read.
func every(book port.Reading, runs []lit.Run) []int {
	held := map[int]bool{}
	var out []int
	for _, one := range runs {
		for _, page := range across(book, one.Start, one.Start+one.Length) {
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
func across(book port.Reading, start, end int) []int {
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
