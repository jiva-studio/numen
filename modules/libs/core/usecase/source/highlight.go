package source

import (
	"context"
	"errors"
	"io/fs"
	"sort"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/internal/correction"
	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

// Highlight is what runs of a source's text say and where they sit on the pages
// they were read from.
//
// Two things place a word — a model recognising a scan, and the document's own text
// layer — and which of them made the text the chunks are places in is the one
// question asked here. It is the question extraction asks when it cuts a source,
// answered by the same column: an offset belongs to one producer's text, and the
// other producer's rectangles cover other words.
//
// A source with no recognition and no layer is lit nowhere, and that is an answer.
type Highlight struct {
	readers vaultport.VaultReaders
	sources sourceport.SourceQueries
	derived vaultport.DerivedStores

	// Documents is optional. It reads a document that carries its own text
	// layer; without one such a document is lit nowhere. A vault whose
	// documents are all recognised needs none.
	Documents sourceport.TextExtractor
}

// A Run is one run of a source's text: what it says, and where it relies on the
// pages it was read from.
type Run struct {
	Text  string
	Boxes []domain.HighlightBox
}

// NewHighlight is what a run of text says and where it relies on the pages it
// was read from: the vault the document is read out of, what says which producer
// made the text the runs are places in, and the store that producer's text and
// coordinates are kept in.
func NewHighlight(
	readers vaultport.VaultReaders, sources sourceport.SourceQueries, derived vaultport.DerivedStores,
) Highlight {
	return Highlight{readers: readers, sources: sources, derived: derived}
}

// Execute is what the runs of one source's text say and where each of them
// sits: the boxes covering it, each on the page it was read from.
//
// asked about a passage and the places around it knows which is which. The text
// and the coordinates are read once however many runs are asked about.
func (u Highlight) Execute(
	ctx context.Context,
	v domain.Vault,
	path string,
	runs []domain.ByteSpan,
) ([]Run, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return nil, err
	}
	// Everything from outside reaches the vault through a reader, so a path
	ref, err := reader.GetFingerprint(ctx, path)
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, nil
	}
	sourceText, exists, err := u.sources.GetSourceText(ctx, v.ID, path)
	if err != nil {
		return nil, err
	}
	// Which producer made a source's text is what says where its offsets are,
	// and a source the index does not hold says nothing about either.
	if !exists {
		return getRuns("", nil, runs), nil
	}
	var store vaultport.DerivedStore
	if u.derived != nil {
		if store, err = u.derived.Open(v); err != nil {
			return nil, err
		}
	}

	// A recognition is of the bytes the index last saw, and its text and
	// coordinates describe those. A file rewritten since is read from its own
	// layer, which is the words that are there now.
	var boxes []domain.HighlightBox
	if sourceText.Producer != "" && ref.IsUnchanged(sourceText.Fingerprint) {
		boxes, err = u.read(ctx, store, sourceText)
	} else {
		sourceText = sourceport.SourceText{}
		boxes, err = u.layer(ctx, reader, ref, runs)
	}
	if err != nil {
		return nil, err
	}
	prose, err := u.prose(ctx, reader, store, path, sourceText)
	if err != nil {
		return nil, err
	}
	return getRuns(prose, boxes, runs), nil
}

// prose is the text the runs are places in: what the producer wrote, or the
// file's own words where no recognition of these bytes stands. A source nothing
// here reads says nothing, and that is an answer.
func (u Highlight) prose(
	ctx context.Context,
	reader vaultport.VaultReader,
	store vaultport.DerivedStore,
	path string,
	sourceText sourceport.SourceText,
) (string, error) {
	of := text.Reader{Vault: reader, Derived: store, Documents: u.Documents}
	doc, err := of.GetDocument(ctx, path, sourceText.Producer, sourceText.Hash)
	if errors.Is(err, text.ErrUnreadable) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return doc.Text, nil
}

// getRuns is what each run says and where it is located, in the order the runs were
// asked about. A run standing nowhere is lit nowhere and keeps its place in the
// answer.
func getRuns(prose string, boxes []domain.HighlightBox, runs []domain.ByteSpan) []Run {
	out := make([]Run, 0, len(runs))
	for _, one := range runs {
		start, length := getRuneBounds(prose, one.From, one.Len())
		out = append(out, Run{
			Text:  prose[start : start+length],
			Boxes: highlight.GetBoxesOver(boxes, one),
		})
	}
	return out
}

// read is where a producer put the words it read. The coordinates are kept
// beside the text they were read with, under the hash of the bytes both came
// from.
func (u Highlight) read(
	ctx context.Context,
	store vaultport.DerivedStore,
	sourceText sourceport.SourceText,
) ([]domain.HighlightBox, error) {
	if store == nil {
		return nil, nil
	}
	raw, err := store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactBoxes, sourceText.Producer, sourceText.Hash))
	if errors.Is(err, fs.ErrNotExist) {
		// The store is a folder on the person's disk and they may empty it.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	boxes := highlight.Unpack(raw)

	// A recognition that was proofread is read with its corrections in it, so a run
	// of its text is a run of the corrected prose and the coordinates say where
	// those words stand.
	corrections, err := store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactCorrections, sourceText.Producer, sourceText.Hash))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return domain.GetCorrectedBoxes(boxes, correction.Unpack(corrections)), nil
}

// layer is where the document's own text layer put the words, over the pages
// the runs fall on and no others.
//
// Which pages those are comes from where each page's text begins, which the
// document says when it is read. A page nothing asked about is not read.
func (u Highlight) layer(
	ctx context.Context,
	reader vaultport.VaultReader,
	ref domain.Fingerprint,
	runs []domain.ByteSpan,
) ([]domain.HighlightBox, error) {
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
	pages := getPagesOfRuns(book, runs)
	if len(pages) == 0 {
		return nil, nil
	}
	return u.Documents.GetHighlights(ctx, raw, book.Pages, pages)
}

// getPagesOfRuns is the pages all the runs fall on, in order and each of them once. Two
// runs on one page are one page read.
func getPagesOfRuns(book domain.TextLayer, runs []domain.ByteSpan) []int {
	target := map[int]bool{}
	var out []int
	for _, one := range runs {
		for _, page := range getPages(book, one.From, one.To) {
			if target[page] {
				continue
			}
			target[page] = true
			out = append(out, page)
		}
	}
	sort.Ints(out)
	return out
}

// getPages is the pages a run of the document's text falls on. A page holds the
// text from where it begins up to where the next page does, and the last page
// holds the rest.
func getPages(book domain.TextLayer, start, end int) []int {
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

// getRuneBounds is a run within the text, standing on whole characters. A run
// beginning past the end of the text is no run at all, a run beginning before
// its start begins at its start instead, and a run of zero or negative length
// is empty.
func getRuneBounds(prose string, start, length int) (int, int) {
	if start < 0 {
		start = 0
	}
	if start >= len(prose) || length <= 0 {
		return min(start, len(prose)), 0
	}
	end := min(start+length, len(prose))
	for start > 0 && !utf8.RuneStart(prose[start]) {
		start--
	}
	for end < len(prose) && !utf8.RuneStart(prose[end]) {
		end++
	}
	return start, end - start
}
