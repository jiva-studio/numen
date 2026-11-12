// Package service is the backend of the embeddings plugin.
package service

import (
	"context"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	hostdomain "github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchsource"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/console"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/engine"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/engine/onnx"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/environment"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/hostmodel"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/mockembedder"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/profilefiles"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/usecase/serving"
)

func newLocalModelOpener(library *onnx.Library) ports.OpenEmbedder {
	// gomutants:disable-next-line RETURN_ZERO reason="production opener wraps engine.Open with native ONNX runtime"
	return func(ctx context.Context, info model.ModelInfo, local model.LocalModel) (ports.Embedder, error) {
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

func newMockEmbedder(logPath string) ports.Embedder {
	return mockembedder.Embedder{LogPath: logPath}
}

func newServer(open ports.OpenModels) *serving.Server {
	return serving.New(serving.Ports{
		Machine:     process.Process{},
		Profiles:    profilefiles.GetDefaultDirectory(),
		Environment: environment.Variables{},
		OpenLocal:   newLocalModelOpener(onnx.NewLibrary()),
		NewMock:     newMockEmbedder,
		OpenModels:  open,
		Report:      console.Report,
	})
}

// Mount registers the search source service on s, with the model the environment configures. The
// model closes with s. The model API, the index and the list of health items of the host are the ones
// the environment names.
func Mount(s *server.Server) error {
	return mount(s, hostmodel.OpenFromEnvironment, getPlaces(host.NewSearchClientFromEnv), health.NewClientFromEnv)
}

func getPlaces(connect func() (*host.SearchClient, error)) openPlace {
	return func(vaultID string) (host.PlaceSearch, error) {
		client, err := connect()
		if err != nil {
			return nil, err
		}
		return client.With(hostdomain.VaultID(vaultID)), nil
	}
}

func mount(s *server.Server, open ports.OpenModels, places openPlace, reports openHealth) error {
	embedSrv := newServer(open)
	s.SetUpdateConfig(embedSrv.Update)
	searchsource.Mount(s, newHandler(embedSrv, places, newSourceHealth(reports)))
	// gomutants:disable-next-line STATEMENT_REMOVE reason="server shutdown callback closes internal embedSrv state"
	s.OnClose(embedSrv.Close)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. The model API of the host is the one
// those variables name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return mount(s, hostmodel.OpenFromVariables(vars), getPlaces(func() (*host.SearchClient, error) {
			return host.NewSearchClientFromVars(vars)
		}), func() (*health.Client, error) { return health.NewClientFromVars(vars) })
	})
}
