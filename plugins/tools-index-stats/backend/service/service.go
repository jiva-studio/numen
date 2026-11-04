// Package service is the backend of the index stats plugin: it reads what the search index holds
// from the host and serves it to the plugin's frontend with the performance profile, the speed of
// embedding and the time of the last change.
package service

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/performance"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/report"
)

// Clock tells the time.
type Clock interface {
	Now() time.Time
}

// Index reads what the search index of the vault the host shows holds.
type Index interface {
	GetIndexStats(ctx context.Context) (host.IndexStats, error)
}

// stats holds what the calls of the frontend share: the host, the meter and the profile.
type stats struct {
	index Index
	clock Clock
	cpus  int
	meter *report.Meter

	mu      sync.Mutex
	profile performance.Profile
}

func newStats(index Index, clk Clock, cpus int) *stats {
	return &stats{
		index: index, clock: clk, cpus: cpus, meter: report.NewMeter(),
		mu: sync.Mutex{}, profile: performance.Balanced,
	}
}

// configure applies the performance profile the host passes with every configuration.
func (s *stats) configure(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.profile = performance.Read(values)
}

func (s *stats) getProfile() performance.Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.profile
}

// get reads the index and answers what it holds.
func (s *stats) get(ctx context.Context, _ api.GetStatsRequest) (api.Stats, error) {
	raw, err := s.index.GetIndexStats(ctx)
	if err != nil {
		return api.Stats{}, convertError(err)
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

// convertError is the failure a client receives for a failed read of the index. An installation with
// no model, or a window with no vault, has no index to show, which is no fault of the call.
func convertError(err error) error {
	switch {
	case errors.Is(err, host.ErrNoModel), errors.Is(err, host.ErrNoVault):
		return server.NewAPIError(server.CodeFailedPrecondition, err.Error())
	case errors.Is(err, host.ErrPermissionDenied):
		return server.NewAPIError(server.CodePermissionDenied, err.Error())
	}
	return fmt.Errorf("read the index statistics: %w", err)
}

// mount registers the service on s.
func mount(s *server.Server, index Index, clk Clock, cpus int) error {
	view := newStats(index, clk, cpus)
	s.SetUpdateConfig(view.configure)
	a := s.NewAPI(api.Service)
	server.Handle(a, api.MethodGetStats, view.get)
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="api.Service and api.MethodGetStats are valid compile-time constants."
	return a.Err()
}

// Mount registers the service on s. The index is the one the host the process was launched for shows.
func Mount(s *server.Server) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	return mount(s, host.NewSearchClient(endpoint), clock.System{}, runtime.NumCPU())
}
