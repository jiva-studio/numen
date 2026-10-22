package source

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetDocumentText reads a run of one document's text, in the same text a search chunk is a place in.
type GetDocumentText struct {
	readers   port.VaultReaders
	sources   port.SourceQueries
	derived   port.DerivedStores
	documents port.TextExtractor
}

func NewGetDocumentText(
	readers port.VaultReaders,
	sources port.SourceQueries,
	derived port.DerivedStores,
	documents port.TextExtractor,
) *GetDocumentText {
	return &GetDocumentText{readers: readers, sources: sources, derived: derived, documents: documents}
}

// DocumentText is a run of a document's text, where it begins, and how much of the
// document stands around it.
type DocumentText struct {
	Path string
	Text string
	// Start and Length are the run that was read, which is what was asked for
	// held within the text.
	Start    int
	Length   int
	Location string
	// Whole is how long the source's text is, so a caller knows whether there
	// is more of it on either side.
	Whole int
}

// MostDocumentText is how much of a document one question carries.
const MostDocumentText = 16000

// Execute reads a run of one document's text.
func (u GetDocumentText) Execute(
	ctx context.Context,
	v domain.Vault,
	path string,
	start, length int,
) (DocumentText, error) {
	res := DocumentText{Path: path, Start: start, Length: length}
	if start < 0 || length <= 0 {
		return res, fmt.Errorf("a run of %s begins at %d and is %d long", path, start, length)
	}
	if length > MostDocumentText {
		return res, fmt.Errorf("read at most %d bytes of a source at a time", MostDocumentText)
	}

	reader, err := u.readers.Open(v)
	if err != nil {
		return res, err
	}
	var store port.DerivedStore
	if u.derived != nil {
		if store, err = u.derived.Open(v); err != nil {
			return res, err
		}
	}

	// Which producer made this source's text is what the index says, and it is
	// the same answer a search sliced its passage at.
	var from, hash string
	if u.sources != nil {
		response, target, err := u.sources.GetSourceText(ctx, v.ID, path)
		if err != nil {
			return res, err
		}
		if target {
			from, hash = response.Producer, response.Hash
		}
	}

	doc, err := text.Reader{Vault: reader, Derived: store, Documents: u.documents}.GetDocument(ctx, path, from, hash)
	if err != nil {
		return res, err
	}

	res.Whole = len(doc.Text)
	res.Start, res.Length = getRuneBounds(doc.Text, start, length)
	if res.Length == 0 {
		return res, nil
	}
	res.Text = doc.Text[res.Start : res.Start+res.Length]
	res.Location = doc.Locate(res.Start)
	return res, nil
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
