package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/internal/adapter/openaiapi"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/completion"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/embedding"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/models"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/reranking"
)

// service is what the plugin builds: the provider of its connection types, the limits its settings
// set, and the gateway whose connections it closes.
type service struct {
	provider *provider
	limits   *limits.Limits
	gateway  *openaiapi.Gateway
}

func newService(source openaiapi.Clock, lists models.Lists) (*service, error) {
	gateway := openaiapi.New(source)
	requestLimits := new(limits.Limits)
	completions, err := completion.New(gateway, requestLimits)
	if err != nil {
		return nil, err
	}
	embeddings, err := embedding.New(gateway, requestLimits)
	if err != nil {
		return nil, err
	}
	rerankings, err := reranking.New(gateway, requestLimits)
	if err != nil {
		return nil, err
	}
	listings, err := models.New(gateway, requestLimits, lists)
	if err != nil {
		return nil, err
	}
	return &service{
		provider: &provider{completion: completions, embedding: embeddings, reranking: rerankings, models: listings},
		limits:   requestLimits,
		gateway:  gateway,
	}, nil
}

// Mount serves the provider on the plugin server and takes the settings of the plugin from Configure. The
// lists of models are kept in the cache directory the host grants, and in memory only where it grants none.
func Mount(srv *server.Server) error {
	dir, err := cache.GetPluginDir()
	if err != nil {
		dir = ""
	}
	built, err := newService(clock.System{}, models.Lists{Dir: dir, Report: openaiapi.ReportListError})
	if err != nil {
		return err
	}
	srv.SetUpdateConfig(built.limits.Configure)
	srv.OnClose(func() error {
		built.gateway.Close()
		return nil
	})
	connection.Mount(srv, built.provider.roles())
	return nil
}
