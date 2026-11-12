// Package indexstats answers what the search index holds, with the performance profile in force,
// the speed of embedding and the time of the last change.
package indexstats

import (
	"context"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/usecase/report"
)

// Clock tells the time.
type Clock interface {
	Now() time.Time
}

// Index reads what the search index of the vault the host shows holds.
type Index interface {
	GetIndexStats(ctx context.Context) (host.IndexStats, error)
}

// Stats holds what the calls share: the index, the meter and the profile.
type Stats struct {
	index Index
	clock Clock
	cpus  int
	meter *report.Meter

	mu      sync.Mutex
	profile performance.Profile
}

// NewStats makes the use case over an index, a clock and the count of processors of the machine.
func NewStats(index Index, clk Clock, cpus int) *Stats {
	return &Stats{
		index: index, clock: clk, cpus: cpus, meter: report.NewMeter(),
		mu: sync.Mutex{}, profile: performance.Balanced,
	}
}

// Configure takes the performance profile from the settings the host supplies.
func (s *Stats) Configure(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.profile = performance.Read(values)
}

func (s *Stats) getProfile() performance.Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.profile
}

// Get reads what the index holds now.
func (s *Stats) Get(ctx context.Context) (api.Stats, error) {
	raw, err := s.index.GetIndexStats(ctx)
	if err != nil {
		return api.Stats{}, err
	}
	derived := s.meter.Observe(s.clock.Now(), report.Counts{Vectors: raw.VectorCount, Chunks: raw.ChunkCount, Sources: raw.SourceCount})
	profile := s.getProfile()
	answer := api.Stats{
		Vectors: raw.VectorCount, Chunks: raw.ChunkCount, Waiting: max(0, raw.ChunkCount-raw.VectorCount),
		Sources: raw.SourceCount, Kinds: report.GroupByKind(raw.Extensions),
		Model: raw.Model, Recipe: raw.Recipe, Dimensions: raw.Dimensions,
		Profile: string(profile), Threads: performance.Resolve(profile, s.cpus).Threads,
		Speed: nil, ChangedAt: "", DatabaseBytes: raw.DatabaseBytes, VectorBytes: raw.VectorBytes,
	}
	if derived.HasSpeed {
		answer.Speed = &derived.Speed
	}
	if derived.HasChanged {
		answer.ChangedAt = derived.ChangedAt.UTC().Format(time.RFC3339)
	}
	return answer, nil
}
