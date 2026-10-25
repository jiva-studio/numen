package highlight_test

import (
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/highlight"
)

const (
	path   = "books/scan.pdf"
	reader = "md.numen.readers.pdf"
)

// newTexts is a vault whose scan has a positioned version made by another plugin: two lines of the
// first page, each in a rectangle of its own.
func newTexts(t *testing.T) *hosttest.Texts {
	t.Helper()
	texts := hosttest.NewTexts("Physics", "md.numen.engines.ocr")
	texts.File(path, []byte("%PDF"))
	provider := hosttest.NewMemoryProvider()
	provider.Hold(path, "layer", []domain.TextPart{
		{Text: "Tidal mills ", Place: domain.TextPlace{Page: 0, Rects: []domain.Rect{{MinX: 0.1, MinY: 0.1, MaxX: 0.9, MaxY: 0.2}}}},
		{Text: "turned with the sea.", Place: domain.TextPlace{Page: 0, Rects: []domain.Rect{{MinX: 0.1, MinY: 0.3, MaxX: 0.9, MaxY: 0.4}}}},
	})
	texts.Provide(reader, provider)
	if _, err := texts.As(reader).PutVersion(t.Context(), host.TextPut{
		Path: path, Key: "layer", Format: domain.TextFormatPositioned, InputHash: domain.HashContent([]byte("%PDF")), Hash: "h",
	}); err != nil {
		t.Fatal(err)
	}
	return texts
}

func read(t *testing.T, s highlight.Service, spans ...*numenv1.Span) (*numenv1.ReadOcrResponse, error) {
	t.Helper()
	res, err := s.ReadOcr(t.Context(), connect.NewRequest(&numenv1.ReadOcrRequest{Path: path, Spans: spans}))
	if err != nil {
		return nil, err
	}
	return res.Msg, nil
}

func TestARunOfTheLiveTextIsLitWhereItStands(t *testing.T) {
	s := highlight.Service{Texts: newTexts(t)}

	got, err := read(t, s, &numenv1.Span{From: 12, To: 18}, &numenv1.Span{From: 0, To: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.GetRuns()) != 2 || got.GetRuns()[0].GetText() != "turned" || got.GetRuns()[1].GetText() != "Tidal" {
		t.Fatalf("runs %v, want each run in the order asked", got.GetRuns())
	}
	boxes := got.GetRuns()[0].GetBoxes()
	if len(boxes) != 1 || boxes[0].GetPage() != 0 || boxes[0].GetRect().GetMinY() != 0.3 || boxes[0].GetSpan().GetFrom() != 12 {
		t.Errorf("the second line stands in %v", boxes)
	}
}

func TestAFileWithNoLiveTextIsLitNowhere(t *testing.T) {
	texts := newTexts(t)
	texts.SetEnabled(reader, false)

	got, err := read(t, highlight.Service{Texts: texts}, &numenv1.Span{From: 0, To: 5})
	if err != nil || len(got.GetRuns()) != 1 || len(got.GetRuns()[0].GetBoxes()) != 0 {
		t.Fatalf("answered %v (%v), want one run lit nowhere", got, err)
	}
}

func TestARunThatIsNoPlaceInTheTextIsRefused(t *testing.T) {
	s := highlight.Service{Texts: newTexts(t)}
	for name, spans := range map[string][]*numenv1.Span{
		"none":            nil,
		"before the text": {{From: -1, To: 3}},
		"empty":           {{From: 3, To: 3}},
		"too many": {
			{From: 0, To: 1}, {From: 0, To: 1}, {From: 0, To: 1}, {From: 0, To: 1}, {From: 0, To: 1},
			{From: 0, To: 1}, {From: 0, To: 1}, {From: 0, To: 1}, {From: 0, To: 1},
		},
	} {
		if _, err := read(t, s, spans...); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s was answered %v", name, err)
		}
	}
}
