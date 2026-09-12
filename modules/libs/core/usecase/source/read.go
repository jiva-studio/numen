package source

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Read is a run of one source's text, as the reader that made that text wrote
// it.
//
// A search answers with a chunk, and a chunk ends where it was cut, which is
// mid-sentence as often as not. This is what stands around it: the same text a
// chunk is a place in, so an offset means one thing to both.
type Read struct {
	Readers port.VaultReaders
	Sources port.SourceQueries
	Derived port.DerivedStores
	// Documents reads a format that needs a library, for a source standing on
	// its own bytes.
	Documents port.TextExtractor
}

// ReadResult is a run of a source's text, where it begins, and how much of the
// source stands around it.
type ReadResult struct {
	Path string
	Text string
	// Start and Length are the run that was read, which is what was asked for
	// held within the text.
	Start  int
	Length int
	// Location is where the run begins, as a person says it.
	Location string
	// Whole is how long the source's text is, so a caller knows whether there
	// is more of it on either side.
	Whole int
}

// MostRead is how much of a source one question carries.
const MostRead = 16000

// Execute reads a run of one source's text.
func (u Read) Execute(
	ctx context.Context,
	v domain.Vault,
	path string,
	start, length int,
) (ReadResult, error) {
	res := ReadResult{Path: path, Start: start, Length: length}
	if start < 0 || length <= 0 {
		return res, fmt.Errorf("a run of %s begins at %d and is %d long", path, start, length)
	}
	if length > MostRead {
		return res, fmt.Errorf("read at most %d bytes of a source at a time", MostRead)
	}

	reader, err := u.Readers.Open(v)
	if err != nil {
		return res, err
	}
	var store port.DerivedStore
	if u.Derived != nil {
		if store, err = u.Derived.Open(v); err != nil {
			return res, err
		}
	}

	// Which producer made this source's text is what the index says, and it is
	// the same answer a search sliced its passage at.
	var from, hash string
	if u.Sources != nil {
		said, held, err := u.Sources.Reading(ctx, v.ID, path)
		if err != nil {
			return res, err
		}
		if held {
			from, hash = said.Producer, said.Hash
		}
	}

	doc, err := text.Reader{Vault: reader, Derived: store, Documents: u.Documents}.Of(ctx, path, from, hash)
	if err != nil {
		return res, err
	}

	res.Whole = len(doc.Text)
	res.Start, res.Length = held(doc.Text, start, length)
	if res.Length == 0 {
		return res, nil
	}
	res.Text = doc.Text[res.Start : res.Start+res.Length]
	res.Location = doc.Locate(res.Start)
	return res, nil
}

// held is a run within the text, standing on whole characters. A run beginning
// past the end of the text is no run at all.
func held(prose string, start, length int) (int, int) {
	if start >= len(prose) {
		return len(prose), 0
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
