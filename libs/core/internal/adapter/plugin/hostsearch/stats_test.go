package hostsearch

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/notes/domain"
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/clock"
	indexport "github.com/jiva-studio/numen/libs/core/usecase/indexing/indexcontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// stubStats answers with the counts it holds and counts the reads.
type stubStats struct {
	counts      indexport.IndexCounts
	sizes       indexport.IndexSizes
	err         error
	countsReads int
	sizesReads  int
	lastVault   string
	lastRecipe  string
}

func (s *stubStats) GetIndexCounts(_ context.Context, vault domain.VaultID, recipe string) (indexport.IndexCounts, error) {
	s.countsReads++
	s.lastVault, s.lastRecipe = string(vault), recipe
	return s.counts, s.err
}

func (s *stubStats) GetIndexSizes(context.Context) (indexport.IndexSizes, error) {
	s.sizesReads++
	return s.sizes, s.err
}

func TestCountsAreCachedForOneSecondAndSizesForHalfAMinute(t *testing.T) {
	source := &stubStats{counts: indexport.IndexCounts{Chunks: 3}, sizes: indexport.IndexSizes{Database: 100}}
	now := clock.NewFake(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC))
	stats := NewStats(source, now)
	ctx := t.Context()

	for range 3 {
		if _, _, err := stats.Get(ctx, "01VAULT", "r"); err != nil {
			t.Fatal(err)
		}
	}
	if source.countsReads != 1 || source.sizesReads != 1 {
		t.Fatalf("three calls in one instant read the counts %d times and the sizes %d times, want once each", source.countsReads, source.sizesReads)
	}

	source.counts = indexport.IndexCounts{Chunks: 4}
	now.Advance(1500 * time.Millisecond)
	counts, _, err := stats.Get(ctx, "01VAULT", "r")
	if err != nil || counts.Chunks != 4 || source.countsReads != 2 || source.sizesReads != 1 {
		t.Errorf("after 1.5 s: counts %+v (%v), read %d and %d times", counts, err, source.countsReads, source.sizesReads)
	}

	source.sizes = indexport.IndexSizes{Database: 200}
	now.Advance(30 * time.Second)
	_, sizes, err := stats.Get(ctx, "01VAULT", "r")
	if err != nil || sizes.Database != 200 || source.sizesReads != 2 {
		t.Errorf("after 31.5 s: sizes %+v (%v), read %d times", sizes, err, source.sizesReads)
	}
}

func TestCountsAreCachedPerVaultAndRecipe(t *testing.T) {
	source := &stubStats{}
	stats := NewStats(source, clock.NewFake(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))
	for _, key := range [][2]string{{"a", "r"}, {"b", "r"}, {"a", "other"}} {
		if _, _, err := stats.Get(t.Context(), domain.VaultID(key[0]), key[1]); err != nil {
			t.Fatal(err)
		}
	}
	if source.countsReads != 3 || source.sizesReads != 1 {
		t.Errorf("read the counts %d times and the sizes %d times, want 3 and 1", source.countsReads, source.sizesReads)
	}
}

func TestAFailedReadIsNotCached(t *testing.T) {
	source := &stubStats{err: errors.New("index closed")}
	stats := NewStats(source, clock.NewFake(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))
	if _, _, err := stats.Get(t.Context(), "v", "r"); !errors.Is(err, source.err) {
		t.Fatalf("got %v", err)
	}
	source.err = nil
	if _, _, err := stats.Get(t.Context(), "v", "r"); err != nil {
		t.Errorf("a read after the failure got %v", err)
	}

	source = &stubStats{}
	stats = NewStats(source, clock.NewFake(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))
	failing := &failingSizes{stubStats: source}
	stats.source = failing
	if _, _, err := stats.Get(t.Context(), "v", "r"); err == nil {
		t.Error("sizes that cannot be read gave no error")
	}
}

type failingSizes struct{ *stubStats }

func (f *failingSizes) GetIndexSizes(context.Context) (indexport.IndexSizes, error) {
	return indexport.IndexSizes{}, errors.New("no dbstat")
}

func TestTheIndexStatisticsCarryTheModelAndTheCounts(t *testing.T) {
	f := newFixture()
	f.search.Stats = NewStats(&stubStats{
		counts: indexport.IndexCounts{
			Chunks: 12, Vectors: 9, Sources: 3,
			Extensions: []indexport.ExtensionCount{{Extension: "md", Sources: 2, Chunks: 7}, {Extension: "pdf", Sources: 1, Chunks: 5}},
		},
		sizes: indexport.IndexSizes{Database: 4096, Vectors: 2048},
	}, clock.NewFake(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))

	got, err := f.search.GetIndexStats(t.Context(), getVault())
	if err != nil {
		t.Fatal(err)
	}
	want := pluginport.PluginIndexStats{
		Recipe: getModel().Recipe(), Model: "test-model", Dimensions: 4,
		Chunks: 12, Vectors: 9, Sources: 3,
		Extensions:    []pluginport.PluginExtensionStats{{Extension: "md", Sources: 2, Chunks: 7}, {Extension: "pdf", Sources: 1, Chunks: 5}},
		DatabaseBytes: 4096, VectorBytes: 2048,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestTheIndexStatisticsAreRefusedWhereThereIsNoModelAndFailWhenTheIndexFails(t *testing.T) {
	f := newFixture()
	failure := errors.New("index closed")
	f.search.Stats = NewStats(&stubStats{err: failure}, clock.NewFake(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))
	if _, err := f.search.GetIndexStats(t.Context(), getVault()); !errors.Is(err, failure) {
		t.Errorf("an index that fails gave %v", err)
	}

	f.search.Embedder = nil
	if _, err := f.search.GetIndexStats(t.Context(), getVault()); !errors.Is(err, pluginport.ErrNoModel) {
		t.Errorf("an installation with no model gave %v", err)
	}
}
