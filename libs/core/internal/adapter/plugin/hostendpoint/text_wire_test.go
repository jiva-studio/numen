package hostendpoint

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestWriteSegmentsCarriesTextHeadingAndSelector(t *testing.T) {
	got := writeSegments([]domain.TextSegment{
		{Text: "one ", Heading: "Intro", Selector: `{"page":3,"boxes":[[0.1,0.2,0.3,0.4]]}`},
		{Text: "two", Selector: `{"t":[1.5,3]}`},
	})
	if len(got) != 2 || got[0].GetText() != "one " || got[0].GetHeading() != "Intro" || got[0].GetSelector() != `{"page":3,"boxes":[[0.1,0.2,0.3,0.4]]}` || got[1].GetText() != "two" {
		t.Fatalf("got %+v", got)
	}
}

func TestWriteVersionCarriesTheSourceKindAndFile(t *testing.T) {
	got := writeVersion(domain.TextVersion{
		Path: "a.mp3", PluginID: "p.x", Key: "k", Source: "q.y/s", Kind: "transcript", File: "k.jsonl", SourceHash: "s", Hash: "h",
	})
	if got.GetId() != "p.x/k" || got.GetSourceVersionId() != "q.y/s" || got.GetKind() != "transcript" || got.GetFile() != "k.jsonl" {
		t.Fatalf("got %+v", got)
	}
}
