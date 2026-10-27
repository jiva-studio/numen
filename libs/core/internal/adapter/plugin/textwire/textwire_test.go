package textwire_test

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/textwire"
)

func TestSegmentsCrossTheWireAndBack(t *testing.T) {
	segments := []domain.TextSegment{
		{Text: "one ", Heading: "Intro", Selector: `{"page":3,"boxes":[[0.1,0.2,0.3,0.4]]}`},
		{Text: "two", Selector: `{"t":[1.5,3]}`},
	}
	if got := textwire.ReadSegments(textwire.WriteSegments(segments)); !reflect.DeepEqual(got, segments) {
		t.Fatalf("got %+v, want %+v", got, segments)
	}
}

func TestWriteVersionCarriesTheSourceKindAndFile(t *testing.T) {
	got := textwire.WriteVersion(domain.TextVersion{
		Path: "a.mp3", PluginID: "p.x", Key: "k", Source: "q.y/s", Kind: "transcript", File: "k.jsonl", SourceHash: "s", Hash: "h",
	})
	if got.GetId() != "p.x/k" || got.GetSourceVersionId() != "q.y/s" || got.GetKind() != "transcript" || got.GetFile() != "k.jsonl" {
		t.Fatalf("got %+v", got)
	}
}
