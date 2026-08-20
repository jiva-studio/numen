package source

import (
	"context"
	"errors"
	"io/fs"
	"sort"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/pdf"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/placed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/text"
)

// Marks says where a run of a source's text sits on the pages it was read from.
//
// Two things place a word — a model reading a scan, and the document's own text
// layer — and which of them made the text the chunks are places in is the one
// question asked here. It is the question extraction asks when it cuts a source,
// answered by the same column: an offset belongs to one producer's text, and the
// other producer's rectangles cover other words.
//
// A source with no reading and no layer is placed nowhere, and that is an
// answer.
type Marks struct {
	Readers port.VaultReaders
	Sources port.SourceQueries
	Derived port.DerivedStores

	// Layer is where the words of some pages of a document sit. It is the
	// document's own answer in the application, and a test puts its own in.
	Layer func(raw []byte, pages []int) ([]placed.Box, error)
}

// Execute is where a run of one source's text sits: the pages it falls on and,
// on each, the rectangles covering it.
func (u Marks) Execute(
	ctx context.Context,
	v domain.Vault,
	path string,
	start, length int,
) ([]placed.Page, error) {
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
	if length <= 0 || start < 0 {
		return nil, nil
	}

	said, held, err := u.Sources.Reading(ctx, v.ID, path)
	if err != nil || !held {
		return nil, err
	}
	// A reading is of the bytes the index last saw, and its coordinates
	// describe those. A file rewritten since is read from its own layer, which
	// is the words that are there now.
	if said.From != "" && ref.Unchanged(domain.FileRef{Size: said.Size, MTime: said.MTime}) {
		return u.read(ctx, v, said, start, length)
	}
	return u.layer(ctx, reader, ref, start, length)
}

// read is where a producer put the words it read. The coordinates are kept
// beside the text they were read with, under the hash of the bytes both came
// from.
func (u Marks) read(
	ctx context.Context,
	v domain.Vault,
	said port.Recognised,
	start, length int,
) ([]placed.Page, error) {
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
	return placed.Marks(placed.Unpack(raw), start, length), nil
}

// layer is where the document's own text layer put the words, over the pages
// the run falls on and no others.
//
// Which pages those are comes from where each page's text begins, which the
// document says when it is read. A page nothing asked about is not read.
func (u Marks) layer(
	ctx context.Context,
	reader port.VaultReader,
	ref domain.FileRef,
	start, length int,
) ([]placed.Page, error) {
	if name, ok := text.ReaderName(ref); !ok || name != text.ReaderPDF {
		// A book made for a screen is set afresh wherever it is shown, and
		// carries no rectangles.
		return nil, nil
	}
	raw, err := reader.Read(ctx, ref.Path)
	if err != nil {
		return nil, err
	}
	book, err := pdf.Read(raw)
	if err != nil {
		return nil, err
	}
	pages := across(book, start, start+length)
	if len(pages) == 0 {
		return nil, nil
	}
	layer := u.Layer
	if layer == nil {
		layer = book.Placed
	}
	boxes, err := layer(raw, pages)
	if err != nil {
		return nil, err
	}
	return placed.Marks(boxes, start, length), nil
}

// across is the pages a run of the document's text falls on. A page holds the
// text from where it begins up to where the next page does, and the last page
// holds the rest.
func across(book *pdf.Book, start, end int) []int {
	if end > len(book.Text) {
		end = len(book.Text)
	}
	if start >= end || len(book.Pages) == 0 {
		return nil
	}
	// The page the run begins on is the last one beginning at or before it.
	first := sort.Search(len(book.Pages), func(i int) bool {
		return book.Pages[i].Offset > start
	}) - 1
	if first < 0 {
		first = 0
	}
	var out []int
	for i := first; i < len(book.Pages) && book.Pages[i].Offset < end; i++ {
		out = append(out, i)
	}
	return out
}
