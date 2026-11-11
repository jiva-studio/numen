package serving

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/console"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/environment"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/mockembedder"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/profilefiles"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/remote"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/loading"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/model"
)

func newEmbeddingServer() *Server {
	return New(Ports{
		Machine:     process.Process{},
		Profiles:    profilefiles.GetDefaultDirectory(),
		Environment: environment.Variables{},
		OpenLocal: func(context.Context, model.ModelInfo, model.LocalModel) (loading.Embedder, error) {
			return mockembedder.Embedder{}, nil
		},
		NewMock:      func(logPath string) loading.Embedder { return mockembedder.Embedder{LogPath: logPath} },
		FetchVectors: remote.FetchVectors,
		Report:       console.Report,
	})
}

func (s *Server) updateConfig(values map[string]string, secrets server.Secrets) {
	key := Secret{Value: "", IsSet: false}
	if list, isCarried := server.ReadSecrets(secrets, server.SecretAPIKey); isCarried {
		key.IsSet = true
		if len(list) > 0 {
			key.Value = list[0].Value
		}
	}
	s.Update(values, key)
}

func isPrecondition(err error) bool {
	var failure PreconditionError
	return errors.As(err, &failure)
}

func encodeJSON(tb testing.TB, w io.Writer, value any) {
	tb.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		tb.Errorf("encode the response: %v", err)
	}
}

func keysOf(values ...string) server.Secrets {
	list := make([]server.Secret, 0, len(values))
	for i, value := range values {
		list = append(list, server.Secret{ID: "s0000000" + string(rune('1'+i)), Name: "Key", Group: "openrouter", Value: value})
	}
	return server.Secrets{server.SecretAPIKey: list}
}
