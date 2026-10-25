package source

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// Highlight is what runs of a source's text say and where they sit on the pages
// of the file.
//
// The runs are places in the text version the index cut the source from, and
// that version's positioned parts are where its words stand. A file rewritten
// since it was cut, or one the index cut from no version, is read from its live
// version. A source with no version, or one whose version has no places, is lit
// nowhere, and that is an answer.
type Highlight struct {
	readers vaultport.VaultReaders
	sources sourceport.SourceQueries
	texts   textport.TextReaders
}

// A Run is one run of a source's text with its boxes on the pages of the file.
type Run struct {
	Text  string
	Boxes []domain.HighlightBox
}

// NewHighlight reads files from the vault readers, looks up which version the
// index extracted a source from in the source queries, and reads the text
// versions of the vault.
func NewHighlight(readers vaultport.VaultReaders, sources sourceport.SourceQueries, texts textport.TextReaders) Highlight {
	return Highlight{readers: readers, sources: sources, texts: texts}
}

// Execute is what the runs of one source's text say and where each of them
// sits, in the order the runs were asked about. The text is read once however
// many runs are asked about.
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
	// the vault does not hold is refused here.
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
	if !exists {
		return getRuns(domain.VersionText{}, runs), nil
	}
	texts, err := u.texts.Open(v)
	if err != nil {
		return nil, err
	}
	doc, err := u.read(ctx, reader, texts, ref, sourceText)
	if errors.Is(err, textport.ErrNoText) {
		return getRuns(domain.VersionText{}, runs), nil
	}
	if err != nil {
		return nil, err
	}
	return getRuns(doc, runs), nil
}

// read is the version the index cut the source from, or the live one where the
// file changed since or the index cut it from none.
func (u Highlight) read(
	ctx context.Context,
	reader vaultport.VaultReader,
	texts textport.Texts,
	ref domain.Fingerprint,
	sourceText sourceport.SourceText,
) (domain.VersionText, error) {
	if sourceText.Producer != "" && ref.IsUnchanged(sourceText.Fingerprint) {
		return texts.GetVersionText(ctx, ref.Path, domain.TextVersionID(sourceText.Producer))
	}
	raw, err := reader.Read(ctx, ref.Path)
	if err != nil {
		return domain.VersionText{}, fmt.Errorf("read %s: %w", ref.Path, err)
	}
	return texts.GetLiveText(ctx, ref.Path, domain.HashContent(raw))
}

// getRuns is what each run says and where it is located, in the order the runs
// were asked about. A run standing nowhere is lit nowhere and keeps its place in
// the answer.
func getRuns(doc domain.VersionText, runs []domain.ByteSpan) []Run {
	out := make([]Run, 0, len(runs))
	for _, one := range runs {
		start, length := getRuneBounds(doc.Text, one.From, one.Len())
		out = append(out, Run{
			Text:  doc.Text[start : start+length],
			Boxes: doc.GetBoxes(one),
		})
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
