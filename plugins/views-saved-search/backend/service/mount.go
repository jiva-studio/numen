// Package service is the backend of the saved search plugin: it builds the host clients, the reranker and
// the settings, and registers saved_search_preview with the plugin server.
package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/internal/adapter/reranker"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/internal/adapter/settings"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/usecase/preview"
)

// Mount registers the tools on s, working the vault through the host the environment names.
func Mount(s *server.Server) error {
	client, err := host.NewClientFromEnv()
	if err != nil {
		return err
	}
	search, err := host.NewSearchClientFromEnv()
	if err != nil {
		return err
	}
	connections, err := connection.NewClientFromEnv()
	if err != nil {
		return err
	}
	return mount(s, client, search, connections)
}

func mount(s *server.Server, client *host.Client, search *host.SearchClient, connections connection.Caller) error {
	current := settings.NewStore()
	rerank := reranker.New(connections)
	open := func(id domain.VaultID) preview.Dependencies {
		return preview.Dependencies{Search: search.With(id), Notes: client.With(id), Reranker: rerank}
	}
	registry, err := NewRegistry(open, current.GetConfig)
	if err != nil {
		return err
	}
	s.SetUpdateConfig(current.Update)
	registry.Mount(s)
	return nil
}
