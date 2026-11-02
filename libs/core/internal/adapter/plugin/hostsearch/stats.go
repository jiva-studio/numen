package hostsearch

import (
	"context"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	indexport "github.com/jiva-studio/numen/libs/core/usecase/indexing/indexcontract"
)

// The ages a stored answer of Stats is used for. The counts change with every change of the index
// and are cheap to read; the sizes walk the pages of the vector tables.
const (
	countsMaxAge = time.Second
	sizesMaxAge  = 30 * time.Second
)

// Clock tells the time.
type Clock interface {
	Now() time.Time
}

type countsKey struct {
	vault  domain.VaultID
	recipe string
}

type storedCounts struct {
	storedAt time.Time
	counts   indexport.IndexCounts
}

// Stats reads what the index holds and keeps each answer for a while, so a view that asks every
// second or two reads the index once for all the windows that ask.
type Stats struct {
	source indexport.IndexStats
	clock  Clock

	mu       sync.Mutex
	counts   map[countsKey]storedCounts
	sizes    indexport.IndexSizes
	sizesAt  time.Time
	hasSizes bool
}

// NewStats reads from source and takes the time from clock.
func NewStats(source indexport.IndexStats, clock Clock) *Stats {
	return &Stats{
		source: source, clock: clock, mu: sync.Mutex{}, counts: map[countsKey]storedCounts{},
		sizes: indexport.IndexSizes{}, sizesAt: time.Time{}, hasSizes: false,
	}
}

// Get returns the counts of the vault under the recipe, at most one second old, and the sizes of the
// index, at most half a minute old. An answer that failed is not kept.
func (s *Stats) Get(ctx context.Context, vaultID domain.VaultID, recipe string) (indexport.IndexCounts, indexport.IndexSizes, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock.Now()
	counts, err := s.getCountsLocked(ctx, now, countsKey{vault: vaultID, recipe: recipe})
	if err != nil {
		return indexport.IndexCounts{}, indexport.IndexSizes{}, err
	}
	sizes, err := s.getSizesLocked(ctx, now)
	if err != nil {
		return indexport.IndexCounts{}, indexport.IndexSizes{}, err
	}
	return counts, sizes, nil
}

func (s *Stats) getCountsLocked(ctx context.Context, now time.Time, key countsKey) (indexport.IndexCounts, error) {
	if kept, ok := s.counts[key]; ok && now.Sub(kept.storedAt) < countsMaxAge {
		return kept.counts, nil
	}
	counts, err := s.source.GetIndexCounts(ctx, key.vault, key.recipe)
	if err != nil {
		return indexport.IndexCounts{}, err
	}
	s.counts[key] = storedCounts{storedAt: now, counts: counts}
	return counts, nil
}

func (s *Stats) getSizesLocked(ctx context.Context, now time.Time) (indexport.IndexSizes, error) {
	if s.hasSizes && now.Sub(s.sizesAt) < sizesMaxAge {
		return s.sizes, nil
	}
	sizes, err := s.source.GetIndexSizes(ctx)
	if err != nil {
		return indexport.IndexSizes{}, err
	}
	s.sizes, s.sizesAt, s.hasSizes = sizes, now, true
	return sizes, nil
}
