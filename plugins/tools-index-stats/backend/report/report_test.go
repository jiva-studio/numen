package report_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/report"
)

func TestTheKindOfASourceIsDerivedFromItsFileExtension(t *testing.T) {
	tests := map[string]report.Kind{
		"md": report.KindNotes, "markdown": report.KindNotes, "txt": report.KindNotes,
		"pdf": report.KindPDF, "epub": report.KindEPUB,
		"mp3": report.KindMedia, "mp4": report.KindMedia, "m4a": report.KindMedia, "webm": report.KindMedia,
		"docx": report.KindOther, "": report.KindOther,
	}
	for extension, want := range tests {
		if got := report.GetKind(extension); got != want {
			t.Errorf("%q is %q, want %q", extension, got, want)
		}
	}
}

func TestExtensionsAreAddedUpByKindInTheFixedOrderAndAKindWithNoSourceIsLeftOut(t *testing.T) {
	got := report.GroupByKind([]host.ExtensionStats{
		{Extension: "docx", SourceCount: 1, ChunkCount: 2},
		{Extension: "mp3", SourceCount: 2, ChunkCount: 40},
		{Extension: "md", SourceCount: 5, ChunkCount: 9},
		{Extension: "markdown", SourceCount: 1, ChunkCount: 1},
		{Extension: "pdf", SourceCount: 0, ChunkCount: 0},
		{Extension: "epub", SourceCount: 1, ChunkCount: 5},
		{Extension: "", SourceCount: 1, ChunkCount: 0},
	})
	want := []report.KindCount{
		{Kind: report.KindNotes, Sources: 6, Chunks: 10},
		{Kind: report.KindEPUB, Sources: 1, Chunks: 5},
		{Kind: report.KindMedia, Sources: 2, Chunks: 40},
		{Kind: report.KindOther, Sources: 2, Chunks: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if empty := report.GroupByKind(nil); len(empty) != 0 {
		t.Errorf("an empty index gave %+v", empty)
	}
}

func getStart() time.Time { return time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC) }

func TestTheSpeedIsTheRiseOfTheVectorsOverTheWindowWhileChunksWait(t *testing.T) {
	meter := report.NewMeter()

	first := meter.Observe(getStart(), report.Counts{Vectors: 100, Chunks: 1000, Sources: 4})
	if first.HasSpeed || first.Speed != 0 {
		t.Errorf("one observation gave the speed %v (%v)", first.Speed, first.HasSpeed)
	}
	soonAt := getStart().Add(500 * time.Millisecond)
	soon := meter.Observe(soonAt, report.Counts{Vectors: 110, Chunks: 1000, Sources: 4})
	if soon.HasSpeed || soon.Speed != 0 || !soon.HasChanged || !soon.ChangedAt.Equal(soonAt) {
		t.Errorf("a span under a second gave %+v", soon)
	}
	atMinSpan := meter.Observe(getStart().Add(time.Second), report.Counts{Vectors: 120, Chunks: 1000, Sources: 4})
	if !atMinSpan.HasSpeed || atMinSpan.Speed != 20 {
		t.Errorf("a span of one second gave %v (%v), want 20", atMinSpan.Speed, atMinSpan.HasSpeed)
	}
	later := meter.Observe(getStart().Add(2*time.Second), report.Counts{Vectors: 140, Chunks: 1000, Sources: 4})
	if !later.HasSpeed || later.Speed != 20 {
		t.Errorf("speed %v (%v), want 20", later.Speed, later.HasSpeed)
	}
}

func TestTheSpeedForgetsObservationsOlderThanTheWindow(t *testing.T) {
	meter := report.NewMeter()
	meter.Observe(getStart(), report.Counts{Vectors: 0, Chunks: 1000, Sources: 1})
	meter.Observe(getStart().Add(9*time.Second), report.Counts{Vectors: 900, Chunks: 1000, Sources: 1})

	atBoundary := meter.Observe(getStart().Add(10*time.Second), report.Counts{Vectors: 900, Chunks: 1000, Sources: 1})
	if !atBoundary.HasSpeed || atBoundary.Speed != 90 {
		t.Errorf("speed at 10 s = %v (%v), want 90", atBoundary.Speed, atBoundary.HasSpeed)
	}

	pastBoundary := meter.Observe(getStart().Add(11*time.Second), report.Counts{Vectors: 920, Chunks: 1000, Sources: 1})
	if !pastBoundary.HasSpeed || pastBoundary.Speed != 10 {
		t.Errorf("speed at 11 s = %v (%v), want 10", pastBoundary.Speed, pastBoundary.HasSpeed)
	}
}

func TestThereIsNoSpeedOnceTheIndexIsEmbeddedOrWhenTheCountFalls(t *testing.T) {
	meter := report.NewMeter()
	meter.Observe(getStart(), report.Counts{Vectors: 500, Chunks: 1000, Sources: 1})

	fell := meter.Observe(getStart().Add(2*time.Second), report.Counts{Vectors: 400, Chunks: 900, Sources: 1})
	if !fell.HasSpeed || fell.Speed != 0 {
		t.Errorf("a fall gave the speed %v (%v), want 0", fell.Speed, fell.HasSpeed)
	}
	embedded := meter.Observe(getStart().Add(4*time.Second), report.Counts{Vectors: 900, Chunks: 900, Sources: 1})
	if embedded.HasSpeed {
		t.Errorf("an embedded index gave the speed %v", embedded.Speed)
	}
	again := meter.Observe(getStart().Add(5*time.Second), report.Counts{Vectors: 900, Chunks: 1000, Sources: 1})
	if again.HasSpeed {
		t.Errorf("a new run measured at once: %v", again.Speed)
	}
}

func TestTheTimeOfTheLastChangeIsTheObservationThatDiffered(t *testing.T) {
	meter := report.NewMeter()

	first := meter.Observe(getStart(), report.Counts{Vectors: 5, Chunks: 5, Sources: 1})
	if first.HasChanged {
		t.Error("the first observation is no change")
	}
	same := meter.Observe(getStart().Add(time.Second), report.Counts{Vectors: 5, Chunks: 5, Sources: 1})
	if same.HasChanged {
		t.Error("the same counts are no change")
	}
	changedAt := getStart().Add(2 * time.Second)
	changed := meter.Observe(changedAt, report.Counts{Vectors: 6, Chunks: 6, Sources: 1})
	if !changed.HasChanged || !changed.ChangedAt.Equal(changedAt) {
		t.Errorf("changed %v at %v, want %v", changed.HasChanged, changed.ChangedAt, changedAt)
	}
	after := meter.Observe(getStart().Add(5*time.Second), report.Counts{Vectors: 6, Chunks: 6, Sources: 1})
	if !after.ChangedAt.Equal(changedAt) {
		t.Errorf("the time moved to %v without a change", after.ChangedAt)
	}
	sources := meter.Observe(getStart().Add(6*time.Second), report.Counts{Vectors: 6, Chunks: 6, Sources: 2})
	if !sources.ChangedAt.Equal(getStart().Add(6 * time.Second)) {
		t.Errorf("a change of the sources alone left the time at %v", sources.ChangedAt)
	}
}
