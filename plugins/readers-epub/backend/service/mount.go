package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// Mount registers the plugin's services on s.
func Mount(s *server.Server) error {
	return mount(s, NewBookServer())
}

func mount(s *server.Server, books *BookServer) error {
	s.OnClose(books.Close)
	s.Register(numenv1connect.NewBookServiceHandler(books, s.HandlerOptions()...))
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. It holds the paths of requests to
// the vault those name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return mount(s, NewBookServerIn(vars["NUMEN_VAULT_PATH"]))
	})
}
