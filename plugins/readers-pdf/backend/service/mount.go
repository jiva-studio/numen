package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/gen/numen/pdf/v1/pdfv1connect"
)

// Mount registers the plugin's services on s, reporting text versions to the host the environment
// names.
func Mount(s *server.Server) error {
	texts, err := host.NewTextClientFromEnv()
	if err != nil {
		return err
	}
	store, err := host.NewStoreClientFromEnv()
	if err != nil {
		return err
	}
	files, err := host.NewClientFromEnv()
	if err != nil {
		return err
	}
	queue, err := host.NewQueueClientFromEnv()
	if err != nil {
		return err
	}
	return mount(s, files, texts, NewTextServer(files, texts, store, queue))
}

func mount(s *server.Server, files host.Vault, texts host.Texts, provider *TextServer) error {
	s.Register(pdfv1connect.NewDocumentServiceHandler(NewDocumentServer(files, texts), s.HandlerOptions()...))
	s.RegisterRoute(pagesPattern, NewPageServer(files))
	s.Register(host.NewTextProviderHandler(provider, s.HandlerOptions()...))
	s.OnClose(provider.Close)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		texts, err := host.NewTextClientFromVars(vars)
		if err != nil {
			return err
		}
		store, err := host.NewStoreClientFromVars(vars)
		if err != nil {
			return err
		}
		files, err := host.NewClientFromVars(vars)
		if err != nil {
			return err
		}
		queue, err := host.NewQueueClientFromVars(vars)
		if err != nil {
			return err
		}
		return mount(s, files, texts, NewTextServer(files, texts, store, queue))
	})
}
