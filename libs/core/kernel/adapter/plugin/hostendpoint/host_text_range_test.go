package hostendpoint

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
)

// textRangeCall is one ReadTextRange call the ranging texts received.
type textRangeCall struct {
	vault  domain.VaultID
	path   string
	id     domain.TextVersionID
	start  int
	length int
}

// rangingTexts are the memory texts whose ranged read of herons.pdf returns two segments of an
// invented text about herons, and that record each ranged read.
type rangingTexts struct {
	*memoryTexts
	ranges []textRangeCall
	pages  int
}

func (r *rangingTexts) ReadText(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, cursor string) (plugin.TextPage, error) {
	r.pages++
	return r.memoryTexts.ReadText(ctx, v, path, id, cursor)
}

func (r *rangingTexts) ReadTextRange(_ context.Context, v domain.Vault, path string, id domain.TextVersionID, start, length int) (plugin.TextRange, error) {
	r.ranges = append(r.ranges, textRangeCall{vault: v.ID, path: path, id: id, start: start, length: length})
	if path != "herons.pdf" {
		return plugin.TextRange{}, text.ErrNoText
	}
	return plugin.TextRange{
		Segments: []domain.TextSegment{
			{Text: "gray heron waits", Heading: "Herons", Selector: `{"page":2}`},
			{Text: " by the reeds", Selector: `{"page":3}`},
		},
		Start: 120,
		Size:  4800,
	}, nil
}

func newRangingTexts(t *testing.T) (*rangingTexts, *callers) {
	t.Helper()
	texts := &rangingTexts{memoryTexts: newMemoryTexts()}
	fakeCallers := newCallers(nil)
	fakeCallers.vault = vaultOf(newMemoryVault(nil))
	fakeCallers.texts = texts
	return texts, fakeCallers
}

func TestRangedReadTextReturnsSegmentsOffsetsAndTextSize(t *testing.T) {
	texts, fakeCallers := newRangingTexts(t)
	client := newTextClient(t, fakeCallers, "token-a")
	length := int64(29)

	res, err := client.ReadText(t.Context(), connect.NewRequest(&v1.ReadTextRequest{
		Path: "herons.pdf", VersionId: "md.numen.a/ocr", Start: 118, Length: &length,
	}))
	if err != nil {
		t.Fatal(err)
	}

	want := textRangeCall{vault: "v1", path: "herons.pdf", id: "md.numen.a/ocr", start: 118, length: 29}
	if len(texts.ranges) != 1 || texts.ranges[0] != want {
		t.Errorf("the texts received the ranged reads %+v, want %+v", texts.ranges, want)
	}
	if texts.pages != 0 {
		t.Errorf("a ranged read read %d pages, want none", texts.pages)
	}
	segments := res.Msg.GetSegments()
	if res.Msg.GetError() != 0 || len(segments) != 2 || res.Msg.GetStart() != 120 || res.Msg.GetSize() != 4800 || res.Msg.GetNextCursor() != "" {
		t.Fatalf("the ranged read returned %v, want two segments from 120 of a text of 4800 bytes", res.Msg)
	}
	if segments[0].GetText() != "gray heron waits" || segments[0].GetHeading() != "Herons" || segments[0].GetSelector() != `{"page":2}` ||
		segments[1].GetText() != " by the reeds" || segments[1].GetSelector() != `{"page":3}` {
		t.Errorf("the segments are %v", segments)
	}
}

func TestARangedReadTextOfLengthZeroPassesZeroForTheRestOfTheText(t *testing.T) {
	texts, fakeCallers := newRangingTexts(t)
	client := newTextClient(t, fakeCallers, "token-a")
	length := int64(0)

	if _, err := client.ReadText(t.Context(), connect.NewRequest(&v1.ReadTextRequest{
		Path: "herons.pdf", VersionId: "md.numen.a/ocr", Start: 4000, Length: &length,
	})); err != nil {
		t.Fatal(err)
	}

	if len(texts.ranges) != 1 || texts.ranges[0].start != 4000 || texts.ranges[0].length != 0 {
		t.Errorf("the texts received the ranged reads %+v, want one from 4000 of length 0", texts.ranges)
	}
}

func TestRangedReadTextOfVersionWithNoTextReturnsMissing(t *testing.T) {
	_, fakeCallers := newRangingTexts(t)
	client := newTextClient(t, fakeCallers, "token-a")
	length := int64(100)

	res, err := client.ReadText(t.Context(), connect.NewRequest(&v1.ReadTextRequest{
		Path: "egrets.pdf", VersionId: "md.numen.a/ocr", Length: &length,
	}))
	if err != nil {
		t.Fatal(err)
	}

	if res.Msg.GetError() != v1.TextErrorCode_TEXT_ERROR_CODE_MISSING || len(res.Msg.GetSegments()) != 0 {
		t.Errorf("a ranged read of a version with no text returned %v, want MISSING", res.Msg)
	}
}

func TestAReadTextWithoutALengthReadsOnePageByCursor(t *testing.T) {
	texts, fakeCallers := newRangingTexts(t)
	client := newTextClient(t, fakeCallers, "token-a")
	if _, err := client.PutTextVersion(t.Context(), connect.NewRequest(&v1.PutTextVersionRequest{
		Path: "herons.pdf", Key: "ocr", Kind: "recognition", InputHash: "f", File: "ocr.jsonl",
	})); err != nil {
		t.Fatal(err)
	}

	res, err := client.ReadText(t.Context(), connect.NewRequest(&v1.ReadTextRequest{
		Path: "herons.pdf", VersionId: "md.numen.a/ocr", Start: 118,
	}))
	if err != nil {
		t.Fatal(err)
	}

	if len(texts.ranges) != 0 || texts.pages != 1 {
		t.Errorf("a read with no length made %d ranged reads and %d page reads, want one page read", len(texts.ranges), texts.pages)
	}
	if segments := res.Msg.GetSegments(); len(segments) != 1 || segments[0].GetText() != "ocr" || res.Msg.GetStart() != 0 || res.Msg.GetSize() != 0 {
		t.Errorf("a read with no length returned %v, want the page of the version", res.Msg)
	}
}
