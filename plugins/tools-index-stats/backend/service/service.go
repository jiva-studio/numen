// Package service is the backend of the index stats plugin: it reads what the search index holds
// from the host and serves it to the plugin's frontend with the performance profile, the speed of
// embedding and the time of the last change.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/internal/adapter/machine"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/usecase/indexstats"
)

func getStats(stats *indexstats.Stats) func(context.Context, api.GetStatsRequest) (api.Stats, error) {
	return func(ctx context.Context, _ api.GetStatsRequest) (api.Stats, error) {
		summary, err := stats.Get(ctx)
		if err != nil {
			return api.Stats{}, convertError(err)
		}
		return toStats(summary), nil
	}
}

func toStats(summary indexstats.Summary) api.Stats {
	kinds := make([]api.KindCount, 0, len(summary.Kinds))
	for _, one := range summary.Kinds {
		kinds = append(kinds, api.KindCount{Kind: api.Kind(one.Kind), Sources: one.Sources, Chunks: one.Chunks})
	}
	return api.Stats{
		Vectors: summary.Vectors, Chunks: summary.Chunks, Waiting: summary.Waiting, Sources: summary.Sources,
		Kinds: kinds, Model: summary.Model, Recipe: summary.Recipe, Dimensions: summary.Dimensions,
		Profile: summary.Profile, Threads: summary.Threads, Speed: summary.Speed, ChangedAt: summary.ChangedAt,
		DatabaseBytes: summary.DatabaseBytes, VectorBytes: summary.VectorBytes,
	}
}

func convertError(err error) error {
	switch {
	case errors.Is(err, host.ErrNoModel), errors.Is(err, host.ErrNoVault):
		return server.NewAPIError(server.CodeFailedPrecondition, err.Error())
	case errors.Is(err, host.ErrPermissionDenied):
		return server.NewAPIError(server.CodePermissionDenied, err.Error())
	}
	return fmt.Errorf("read the index statistics: %w", err)
}

func mount(s *server.Server, stats *indexstats.Stats) error {
	s.SetUpdateConfig(func(values map[string]string) { stats.Configure(values) })
	a := s.NewAPI(api.Service)
	server.Handle(a, api.MethodGetStats, getStats(stats))
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="api.Service and api.MethodGetStats are valid compile-time constants."
	return a.Err()
}

// Mount registers the service on s. The index is the one the host the process was launched for shows.
func Mount(s *server.Server) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	return mount(s, indexstats.NewStats(host.NewSearchClient(endpoint), clock.System{}, machine.GetCPUCount()))
}
