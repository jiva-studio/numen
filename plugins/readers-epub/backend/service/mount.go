package service

import (
	"net/http"
	"os"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/gen/numen/epub/v1/epubv1connect"
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
	defaultRoot := func() string { return os.Getenv("NUMEN_VAULT_PATH") }
	return mount(s, NewBookServer(files, defaultRoot), NewTextServer(files, texts, store, queue))
}

func mount(s *server.Server, books *BookServer, texts *TextServer) error {
	s.OnClose(books.Close)
	s.OnClose(texts.Close)
	s.Register(epubv1connect.NewBookServiceHandler(books, s.HandlerOptions()...))
	s.RegisterRoute(entriesPattern, NewEntryServer(books))
	s.Register(host.NewTextProviderHandler(texts, s.HandlerOptions()...))
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. A request naming no vault works the
// vault those name, and text versions are reported to the host they name.
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
		root := vars["NUMEN_VAULT_PATH"]
		defaultRoot := func() string { return root }
		return mount(s, NewBookServer(files, defaultRoot), NewTextServer(files, texts, store, queue))
	})
}
