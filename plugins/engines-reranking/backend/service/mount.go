// Package service is the backend of the reranking plugin: it builds the reranking use case over the ONNX
// engine and serves it as the provider of the local-reranker connection type.
package service

import (
	"context"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/internal/adapter/console"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/internal/adapter/engine"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/internal/adapter/profilefiles"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/ports"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/usecase/reranking"
)

func newScorerOpener(library *onnx.Library, fetch engine.Fetch, queue host.InferenceQueue) ports.OpenScorer {
	return func(ctx context.Context, p profile.Profile) (ports.Scorer, error) {
		scorer, err := engine.Open(ctx, library, p, fetch, queue)
		if err != nil {
			return nil, err
		}
		return scorer, nil
	}
}

func openRerankingUseCase(queue host.InferenceQueue, fetch engine.Fetch) (*reranking.Reranker, error) {
	return reranking.New(profilefiles.GetDefaultDirectory(), newScorerOpener(onnx.NewLibrary(), fetch, queue), queue, console.Report)
}

// Mount serves the provider of the local-reranker connection type on s. The loaded model closes with s.
func Mount(s *server.Server) error {
	r, err := openRerankingUseCase(host.NewInferenceQueue(s.GetHostEndpoint), getModelFetch(s))
	if err != nil {
		return err
	}
	s.AddCloser(r.Close)
	connection.Mount(s, newProvider(r))
	return nil
}

func getModelFetch(s *server.Server) engine.Fetch {
	return engine.Fetch{
		Dependency: func(ctx context.Context, id string) (string, error) {
			endpoint, err := s.GetHostEndpoint()
			if err != nil {
				return "", err
			}
			return host.NewDependencyClient(endpoint).FetchDependency(ctx, id)
		},
		File: func(ctx context.Context, address, sum string) (string, error) {
			endpoint, err := s.GetHostEndpoint()
			if err != nil {
				return "", err
			}
			return host.NewDependencyClient(endpoint).FetchModelFile(ctx, address, sum)
		},
	}
}
