package textwire_test

import (
	"reflect"
	"testing"
	"time"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/textwire"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

func TestEveryFormatCrossesTheWireAndBack(t *testing.T) {
	for _, f := range domain.TextFormats() {
		if got := textwire.ReadFormat(textwire.WriteFormat(f)); got != f {
			t.Errorf("%s came back as %q", f, got)
		}
	}
	if got := textwire.ReadFormat(v1.TextFormat_TEXT_FORMAT_UNSPECIFIED); got != "" {
		t.Errorf("no format came back as %q", got)
	}
	testsupport.RoundTrip(t, textwire.ReadFormat, textwire.WriteFormat)
}

func TestPartsCrossTheWireAndBack(t *testing.T) {
	parts := []domain.TextPart{
		{Text: "one ", Section: "Intro", Place: domain.TextPlace{Page: 2, Rects: []domain.Rect{{MinX: 0.1, MinY: 0.2, MaxX: 0.3, MaxY: 0.4}}}},
		{Text: "two", Place: domain.TextPlace{Start: 1500 * time.Millisecond, End: 3 * time.Second}},
	}
	got := textwire.ReadParts(textwire.WriteParts(parts))
	want := parts
	want[1].Place.Rects = []domain.Rect{}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
