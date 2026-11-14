// Package service is the backend of the reranking plugin: it builds the reranking use case over the ONNX
// engine and serves it as the provider of the local-reranker connection type.
package service

import (
	"context"

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

func newScorerOpener(library *onnx.Library) ports.OpenScorer {
	return func(ctx context.Context, p profile.Profile) (ports.Scorer, error) {
		scorer, err := engine.Open(ctx, library, p)
		if err != nil {
			return nil, err
		}
		return scorer, nil
	}
}

func openRerankingUseCase() (*reranking.Reranking, error) {
	return reranking.New(profilefiles.GetDefaultDirectory(), newScorerOpener(onnx.NewLibrary()), console.Report)
}

// Mount serves the provider of the local-reranker connection type on s. The loaded model closes with s.
func Mount(s *server.Server) error {
	r, err := openRerankingUseCase()
	if err != nil {
		return err
	}
	s.OnClose(r.Close)
	connection.Mount(s, newProvider(r))
	return nil
}
