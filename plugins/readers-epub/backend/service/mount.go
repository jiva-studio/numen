package service

import (
	"net/http"
	"os"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// Mount registers the plugin's services on s, reporting text versions to the host the environment
// names.
func Mount(s *server.Server) error {
	texts, err := host.NewTextClientFromEnv()
	if err != nil {
		return err
	}
	vaultRoot := func() string { return os.Getenv("NUMEN_VAULT_PATH") }
	return mount(s, NewBookServer(), NewTextServer(vaultRoot, texts))
}

func mount(s *server.Server, books *BookServer, texts *TextServer) error {
	s.OnClose(books.Close)
	s.Register(numenv1connect.NewBookServiceHandler(books, s.HandlerOptions()...))
	s.Register(host.NewTextProviderHandler(texts, s.HandlerOptions()...))
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. It holds the paths of requests to
// the vault those name, and reports text versions to the host they name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		texts, err := host.NewTextClientFromVars(vars)
		if err != nil {
			return err
		}
		root := vars["NUMEN_VAULT_PATH"]
		return mount(s, NewBookServerIn(root), NewTextServer(func() string { return root }, texts))
	})
}
