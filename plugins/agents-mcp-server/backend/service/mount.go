package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/usecase/exposetools"
	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/usecase/hostserver"
)

// Mount registers the plugin's services and handlers on the plugin server.
func Mount(s *server.Server) error {
	var client exposetools.Host
	if endpoint, err := hostcall.FromEnv(); err == nil {
		client = tools.NewClient(endpoint)
	}

	srv := newEndpoint("", client)

	s.OnClose(func() error {
		// gomutants:disable-next-line RETURN_ERROR_NIL reason="loopback listener shutdown within the timeout does not error"
		return srv.Close(context.Background())
	})

	s.SetConfigure(func(ctx context.Context, values map[string]string) error {
		if !srv.host.Aggregator.HasClient() {
			if endpoint, err := hostcall.FromVars(values); err == nil {
				srv.host.Aggregator.SetClient(tools.NewClient(endpoint))
			} else if endpoint, err := hostcall.FromEnv(); err == nil {
				srv.host.Aggregator.SetClient(tools.NewClient(endpoint))
			}
		}

		srv.SetToken(hostserver.ParseSettings(values).Token)
		srv.host.Sync(ctx)
		return nil
	})

	s.Register("/mcp", srv.Handler())
	s.Register("/mcp/", srv.Handler())
	s.Register("/md.numen.agents.mcp.server/", srv.Handler())

	ln, err := s.OpenListener("mcp")
	if err != nil {
		return err
	}
	return srv.Serve(ln)
}
