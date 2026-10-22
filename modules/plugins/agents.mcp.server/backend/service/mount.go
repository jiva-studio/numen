package service

import (
	"context"
	"net/http"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// Mount registers the plugin's services and handlers on the plugin server.
func Mount(s *server.Server) error {
	var client ToolClient
	if endpoint, err := hostcall.FromEnv(); err == nil {
		client = numenv1connect.NewPluginToolServiceClient(hostcall.HTTPClient(), endpoint.BaseURL, endpoint.Options()...)
	}

	cfg := readConfig(nil)
	srv, err := NewServer(cfg, client)
	if err != nil {
		return err
	}

	s.OnClose(func() {
		_ = srv.Close(context.Background())
	})

	s.SetConfigure(func(ctx context.Context, values map[string]string) error {
		if srv.aggregator.client == nil {
			if endpoint, err := hostcall.FromVars(values); err == nil {
				srv.aggregator.client = numenv1connect.NewPluginToolServiceClient(
					hostcall.HTTPClient(),
					endpoint.BaseURL,
					endpoint.Options()...,
				)
			} else if endpoint, err := hostcall.FromEnv(); err == nil {
				srv.aggregator.client = numenv1connect.NewPluginToolServiceClient(
					hostcall.HTTPClient(),
					endpoint.BaseURL,
					endpoint.Options()...,
				)
			}
		}

		newCfg := readConfig(values)
		if newCfg.VaultRoot != "" {
			srv.config.VaultRoot = newCfg.VaultRoot
		}
		if newCfg.DiscoveryPath != "" {
			srv.config.DiscoveryPath = newCfg.DiscoveryPath
		}
		if newCfg.Token != "" {
			srv.config.Token = newCfg.Token
		}

		srv.writeDiscoveryInfo()
		_ = srv.aggregator.SyncTools(ctx)
		return nil
	})

	s.Register("/sse", srv.Handler())
	s.Register("/sse/", srv.Handler())
	s.Register("/mcp", srv.Handler())
	s.Register("/mcp/", srv.Handler())
	s.Register("/messages", srv.Handler())
	s.Register("/messages/", srv.Handler())
	s.Register("/numen.v1.McpServerService/", srv.Handler())

	return srv.Start()
}

// NewHandler returns the handler of the plugin backend for a host that links the plugin in.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
