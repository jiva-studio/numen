package hostsearch

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/index"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// stubStats answers with the counts it holds and counts the reads.
type stubStats struct {
	counts      index.Counts
	sizes       index.Sizes
	err         error
	countsReads int
	sizesReads  int
	lastVault   string
	lastRecipe  string
}

func (s *stubStats) GetIndexCounts(_ context.Context, vault domain.VaultID, recipe string) (index.Counts, error) {
	s.countsReads++
	s.lastVault, s.lastRecipe = string(vault), recipe
	return s.counts, s.err
}

func (s *stubStats) GetIndexSizes(context.Context) (index.Sizes, error) {
	s.sizesReads++
	return s.sizes, s.err
}

func TestStatsCachesCountsForOneSecondAndSizesForHalfAMinute(t *testing.T) {
	source := &stubStats{counts: index.Counts{Chunks: 3}, sizes: index.Sizes{Database: 100}}
	now := testsupport.NewClockAt(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC))
	stats := search.NewStats(source, now)
	ctx := t.Context()

	for range 3 {
		if _, _, err := stats.Get(ctx, "01VAULT", "r"); err != nil {
			t.Fatal(err)
		}
	}
	if source.countsReads != 1 || source.sizesReads != 1 {
		t.Fatalf("three calls in one instant read the counts %d times and the sizes %d times, want once each", source.countsReads, source.sizesReads)
	}

	source.counts = index.Counts{Chunks: 4}
	now.Advance(1500 * time.Millisecond)
	counts, _, err := stats.Get(ctx, "01VAULT", "r")
	if err != nil || counts.Chunks != 4 || source.countsReads != 2 || source.sizesReads != 1 {
		t.Errorf("after 1.5 s: counts %+v (%v), read %d and %d times", counts, err, source.countsReads, source.sizesReads)
	}

	source.sizes = index.Sizes{Database: 200}
	now.Advance(30 * time.Second)
	_, sizes, err := stats.Get(ctx, "01VAULT", "r")
	if err != nil || sizes.Database != 200 || source.sizesReads != 2 {
		t.Errorf("after 31.5 s: sizes %+v (%v), read %d times", sizes, err, source.sizesReads)
	}
}

func TestStatsCachesCountsPerVaultAndRecipe(t *testing.T) {
	source := &stubStats{}
	stats := search.NewStats(source, testsupport.NewClockAt(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))
	for _, key := range [][2]string{{"a", "r"}, {"b", "r"}, {"a", "other"}} {
		if _, _, err := stats.Get(t.Context(), domain.VaultID(key[0]), key[1]); err != nil {
			t.Fatal(err)
		}
	}
	if source.countsReads != 3 || source.sizesReads != 1 {
		t.Errorf("read the counts %d times and the sizes %d times, want 3 and 1", source.countsReads, source.sizesReads)
	}
}

func TestStatsDoesNotCacheFailedRead(t *testing.T) {
	source := &stubStats{err: errors.New("index closed")}
	stats := search.NewStats(source, testsupport.NewClockAt(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))
	if _, _, err := stats.Get(t.Context(), "v", "r"); !errors.Is(err, source.err) {
		t.Fatalf("got %v", err)
	}
	source.err = nil
	if _, _, err := stats.Get(t.Context(), "v", "r"); err != nil {
		t.Errorf("a read after the failure got %v", err)
	}

	source = &stubStats{}
	failing := &failingSizes{stubStats: source}
	stats = search.NewStats(failing, testsupport.NewClockAt(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))
	if _, _, err := stats.Get(t.Context(), "v", "r"); err == nil {
		t.Error("sizes that cannot be read gave no error")
	}
}

type failingSizes struct{ *stubStats }

func (f *failingSizes) GetIndexSizes(context.Context) (index.Sizes, error) {
	return index.Sizes{}, errors.New("no dbstat")
}

func TestGetIndexStatsCarriesTheModelAndTheCounts(t *testing.T) {
	f := newFixture()
	f.search.Stats = search.NewStats(&stubStats{
		counts: index.Counts{
			Chunks: 12, Vectors: 9, Sources: 3,
			Extensions: []index.ExtensionCount{{Extension: "md", Sources: 2, Chunks: 7}, {Extension: "pdf", Sources: 1, Chunks: 5}},
		},
		sizes: index.Sizes{Database: 4096, Vectors: 2048},
	}, testsupport.NewClockAt(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))

	got, err := f.search.GetIndexStats(t.Context(), getVault())
	if err != nil {
		t.Fatal(err)
	}
	want := plugin.IndexStats{
		Recipe: getModel().Recipe(), Model: "test-model", Dimensions: 4,
		Chunks: 12, Vectors: 9, Sources: 3,
		Extensions:    []plugin.ExtensionStats{{Extension: "md", Sources: 2, Chunks: 7}, {Extension: "pdf", Sources: 1, Chunks: 5}},
		DatabaseBytes: 4096, VectorBytes: 2048,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestGetIndexStatsRefusesNoModelAndFailsWhenIndexFails(t *testing.T) {
	f := newFixture()
	failure := errors.New("index closed")
	f.search.Stats = search.NewStats(&stubStats{err: failure}, testsupport.NewClockAt(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))
	if _, err := f.search.GetIndexStats(t.Context(), getVault()); !errors.Is(err, failure) {
		t.Errorf("an index that fails gave %v", err)
	}

	f.search.Embedder = nil
	if _, err := f.search.GetIndexStats(t.Context(), getVault()); !errors.Is(err, plugin.ErrNoModel) {
		t.Errorf("an installation with no model gave %v", err)
	}
}
