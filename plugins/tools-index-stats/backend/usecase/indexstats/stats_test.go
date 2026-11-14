package indexstats

import (
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/report"
)

const recipe = "local|bge-m3|1024|512|mean|int8@0.4"

func getStart() time.Time { return time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC) }

func newIndex() *hosttest.Search {
	search := hosttest.NewSearch("physics")
	search.SetModel(hosttest.SearchModel{Name: "bge-m3", Recipe: recipe, FloorHint: 0})
	search.SetProgress(13100, 12480)
	search.SetIndexStats(host.IndexStats{
		Recipe: "", Model: "", Dimensions: 1024, ChunkCount: 0, VectorCount: 0,
		SourceCount: 84,
		Extensions: []host.ExtensionStats{
			{Extension: "md", SourceCount: 70, ChunkCount: 4100},
			{Extension: "pdf", SourceCount: 9, ChunkCount: 7000},
			{Extension: "epub", SourceCount: 5, ChunkCount: 2000},
		},
		DatabaseBytes: 73400320, VectorBytes: 52428800,
	})
	return search
}

func TestTheStatsCarryTheCountsTheModelAndTheProfile(t *testing.T) {
	view := NewStats(newIndex(), newFakeClock(getStart()), 16)
	view.Configure(map[string]string{"host.performanceProfile": "maximum"})

	got, err := view.Get(t.Context())

	if err != nil {
		t.Fatal(err)
	}
	want := Summary{
		Vectors: 12480, Chunks: 13100, Waiting: 620, Sources: 84,
		Kinds: []report.KindCount{
			{Kind: report.KindNotes, Sources: 70, Chunks: 4100},
			{Kind: report.KindPDF, Sources: 9, Chunks: 7000},
			{Kind: report.KindEPUB, Sources: 5, Chunks: 2000},
		},
		Model: "bge-m3", Recipe: recipe, Dimensions: 1024,
		Profile: "maximum", Threads: 8, Speed: nil, ChangedAt: "",
		DatabaseBytes: 73400320, VectorBytes: 52428800,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestTheProfileIsBalancedUntilTheHostConfiguresThePlugin(t *testing.T) {
	view := NewStats(newIndex(), newFakeClock(getStart()), 16)

	got, err := view.Get(t.Context())

	if err != nil || got.Profile != "balanced" || got.Threads != 4 {
		t.Errorf("profile %q with %d threads (%v), want balanced with 4", got.Profile, got.Threads, err)
	}
}

func TestTheSpeedAndTheTimeOfTheLastChangeFollowTheCallsOfTheView(t *testing.T) {
	index := newIndex()
	now := newFakeClock(getStart())
	view := NewStats(index, now, 8)
	ctx := t.Context()

	first, err := view.Get(ctx)
	if err != nil || first.Speed != nil || first.ChangedAt != "" {
		t.Fatalf("first call: speed %v, changed at %q (%v)", first.Speed, first.ChangedAt, err)
	}

	now.Advance(2 * time.Second)
	index.SetProgress(13100, 12520)
	second, err := view.Get(ctx)
	if err != nil || second.Speed == nil || *second.Speed != 20 || second.ChangedAt != "2026-11-02T12:00:02Z" {
		t.Errorf("second call: speed %v, changed at %q (%v), want 20 and 12:00:02", second.Speed, second.ChangedAt, err)
	}

	now.Advance(time.Second)
	index.SetProgress(13100, 13100)
	done, err := view.Get(ctx)
	if err != nil || done.Speed != nil || done.Waiting != 0 || done.ChangedAt != "2026-11-02T12:00:03Z" {
		t.Errorf("embedded: speed %v, waiting %d, changed at %q (%v)", done.Speed, done.Waiting, done.ChangedAt, err)
	}
}

func TestAnIndexWithMoreVectorsThanChunksWaitsForNothing(t *testing.T) {
	index := newIndex()
	index.SetProgress(10, 12)

	got, err := NewStats(index, newFakeClock(getStart()), 8).Get(t.Context())

	if err != nil || got.Waiting != 0 {
		t.Errorf("waiting %d (%v), want 0", got.Waiting, err)
	}
}

func newFakeClock(start time.Time) *clock.Fake {
	source := new(clock.Fake)
	source.Set(start)
	return source
}
