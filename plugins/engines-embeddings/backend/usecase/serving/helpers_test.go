package serving

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/console"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/environment"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/mockembedder"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/profilefiles"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
)

func newEmbeddingServer() *Server {
	return newEmbeddingServerWith(nil)
}

func newEmbeddingServerWith(open ports.OpenModels) *Server {
	return New(Ports{
		Machine:     process.Process{},
		Profiles:    profilefiles.GetDefaultDirectory(),
		Environment: environment.Variables{},
		OpenLocal: func(context.Context, model.ModelInfo, model.LocalModel) (ports.Embedder, error) {
			return mockembedder.Embedder{}, nil
		},
		NewMock:    func(logPath string) ports.Embedder { return mockembedder.Embedder{LogPath: logPath} },
		OpenModels: open,
		Report:     console.Report,
	})
}

func (s *Server) updateConfig(values map[string]string) {
	s.Update(values)
}

func isPrecondition(err error) bool {
	var failure PreconditionError
	return errors.As(err, &failure)
}

func getModelInfoOf(t *testing.T, srv *Server) ModelInfo {
	t.Helper()
	info, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return info
}
