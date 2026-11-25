package search

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/index"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// The ages a stored answer of Stats is used for. The counts change with every change of the index
// and are cheap to read; the sizes walk the pages of the vector tables.
const (
	countsMaxAge = time.Second
	sizesMaxAge  = 30 * time.Second
)

// IndexStats reads what the index holds. It only reads, so it never waits for a writer.
type IndexStats interface {
	// GetIndexCounts returns the counts of the vault. A vault the index does not know has none.
	GetIndexCounts(ctx context.Context, vaultID domain.VaultID, recipe string) (index.Counts, error)
	// GetIndexSizes returns the sizes of the index.
	GetIndexSizes(ctx context.Context) (index.Sizes, error)
}

type countsKey struct {
	vault  domain.VaultID
	recipe string
}

type storedCounts struct {
	storedAt time.Time
	counts   index.Counts
}

// Stats reads what the index holds and keeps each answer for a while, so a view that asks every
// second or two reads the index once for all the windows that ask.
type Stats struct {
	source IndexStats
	clock  machineport.Clock

	mu       sync.Mutex
	counts   map[countsKey]storedCounts
	sizes    index.Sizes
	sizesAt  time.Time
	hasSizes bool
}

// NewStats reads from source and takes the time from clock.
func NewStats(source IndexStats, clock machineport.Clock) *Stats {
	return &Stats{
		source: source, clock: clock, mu: sync.Mutex{}, counts: map[countsKey]storedCounts{},
		sizes: index.Sizes{}, sizesAt: time.Time{}, hasSizes: false,
	}
}

// StatsSnapshot holds the index counts for one vault and recipe along with the index sizes.
type StatsSnapshot struct {
	Counts index.Counts
	Sizes  index.Sizes
}

// Get returns the counts of the vault under the recipe, at most one second old, and the sizes of the
// index, at most half a minute old. An answer that failed is not kept.
func (s *Stats) Get(ctx context.Context, vaultID domain.VaultID, recipe string) (StatsSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock.Now()
	counts, err := s.getCountsLocked(ctx, now, countsKey{vault: vaultID, recipe: recipe})
	if err != nil {
		return StatsSnapshot{}, err
	}
	sizes, err := s.getSizesLocked(ctx, now)
	if err != nil {
		return StatsSnapshot{}, err
	}
	return StatsSnapshot{Counts: counts, Sizes: sizes}, nil
}

// Execute reads what the index holds for the vault and the space it takes, with the model in use.
func (s *Stats) Execute(ctx context.Context, v domain.Vault, model searchsource.Description) (plugin.IndexStats, error) {
	recipe := model.Recipe()
	snapshot, err := s.Get(ctx, v.ID, recipe)
	if err != nil {
		return plugin.IndexStats{}, fmt.Errorf("read the index statistics of %s: %w", v.ID, err)
	}
	extensions := make([]plugin.ExtensionStats, len(snapshot.Counts.Extensions))
	for i, one := range snapshot.Counts.Extensions {
		extensions[i] = plugin.ExtensionStats{Extension: one.Extension, Sources: one.Sources, Chunks: one.Chunks}
	}
	return plugin.IndexStats{
		Recipe: recipe, Model: model.Name, Dimensions: int64(model.Width),
		Chunks: snapshot.Counts.Chunks, Vectors: snapshot.Counts.Vectors,
		Sources: snapshot.Counts.Sources, Extensions: extensions,
		DatabaseBytes: snapshot.Sizes.Database, VectorBytes: snapshot.Sizes.Vectors,
	}, nil
}

func (s *Stats) getCountsLocked(ctx context.Context, now time.Time, key countsKey) (index.Counts, error) {
	if kept, ok := s.counts[key]; ok && now.Sub(kept.storedAt) < countsMaxAge {
		return kept.counts, nil
	}
	counts, err := s.source.GetIndexCounts(ctx, key.vault, key.recipe)
	if err != nil {
		return index.Counts{}, err
	}
	s.counts[key] = storedCounts{storedAt: now, counts: counts}
	return counts, nil
}

func (s *Stats) getSizesLocked(ctx context.Context, now time.Time) (index.Sizes, error) {
	if s.hasSizes && now.Sub(s.sizesAt) < sizesMaxAge {
		return s.sizes, nil
	}
	sizes, err := s.source.GetIndexSizes(ctx)
	if err != nil {
		return index.Sizes{}, err
	}
	s.sizes, s.sizesAt, s.hasSizes = sizes, now, true
	return sizes, nil
}
