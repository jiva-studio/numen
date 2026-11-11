package serving

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/console"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/environment"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/profilefiles"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/remote"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/loading"
)

func newEmbeddingServer() *Server {
	return New(Ports{
		Machine:     process.Process{},
		Profiles:    profilefiles.GetDefaultDirectory(),
		Environment: environment.Variables{},
		OpenLocal: func(context.Context, engine.ModelInfo, engine.LocalModel) (loading.Embedder, error) {
			return engine.MockEmbedder{}, nil
		},
		FetchVectors: remote.FetchVectors,
		Report:       console.Report,
	})
}

// updateConfig applies the values and the secrets the way the host delivers them.
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
	var refusal PreconditionError
	return errors.As(err, &refusal)
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
