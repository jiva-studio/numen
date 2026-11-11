// Package service is the backend of the embeddings plugin.
package service

import (
	"context"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/console"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/environment"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/profilefiles"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/remote"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/loading"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/onnx"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/serving"
)

func newLocalModelOpener(library *onnx.Library) serving.OpenEmbedder {
	// gomutants:disable-next-line RETURN_ZERO reason="production opener wraps engine.Open with native ONNX runtime"
	return func(ctx context.Context, info engine.ModelInfo, local engine.LocalModel) (loading.Embedder, error) {
		emb, err := engine.Open(ctx, library, info, local, nil)
		// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="production opener wraps engine.Open with native ONNX runtime"
		if err != nil {
			// gomutants:disable-next-line RETURN_ERROR_NIL reason="production opener wraps engine.Open with native ONNX runtime"
			return nil, err
		}
		// gomutants:disable-next-line RETURN_ZERO reason="production opener wraps engine.Open with native ONNX runtime"
		return emb, nil
	}
}

func newServer() *serving.Server {
	return serving.New(serving.Ports{
		Machine:      process.Process{},
		Profiles:     profilefiles.GetDefaultDirectory(),
		Environment:  environment.Variables{},
		OpenLocal:    newLocalModelOpener(onnx.NewLibrary()),
		FetchVectors: remote.FetchVectors,
		Report:       console.Report,
	})
}

// Mount registers the embedding service on s, with the model the environment configures. The
// embedder closes with s.
func Mount(s *server.Server) error {
	embedSrv := newServer()
	s.SetUpdateConfig(updateConfig(embedSrv))
	embedding.Mount(s, handler{model: embedSrv})
	// gomutants:disable-next-line STATEMENT_REMOVE reason="server shutdown callback closes internal embedSrv state"
	s.OnClose(embedSrv.Close)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
