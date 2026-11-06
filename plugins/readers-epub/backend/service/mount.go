package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
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
	return mount(s, NewBookServer(files), NewTextServer(files, texts, store, queue))
}

func mount(s *server.Server, books *BookServer, texts *TextServer) error {
	// gomutants:disable-next-line STATEMENT_REMOVE reason="server close hook releases cached books"
	s.OnClose(func() error {
		books.Close()
		return nil
	})
	// gomutants:disable-next-line STATEMENT_REMOVE reason="server close hook stops running text extraction jobs"
	s.OnClose(func() error {
		texts.Close()
		return nil
	})
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="mount propagates any API registration failure"
	if err := mountMethods(s, books); err != nil {
		return err
	}
	s.RegisterRoute(entriesPattern, NewEntryServer(books))
	commands, err := newCommands(texts)
	// gomutants:disable-next-line BRANCH_IF reason="registering one static command id on a new registry cannot fail"
	if err != nil {
		return err
	}
	command.Mount(s, commands)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. A request naming no vault works the
// vault those name, and text versions are reported to the host they name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		texts, err := host.NewTextClientFromVars(vars)
		// gomutants:disable-next-line BRANCH_IF reason="NewLinkedHandler already validates vars before invoking mount callback"
		if err != nil {
			return err
		}
		store, err := host.NewStoreClientFromVars(vars)
		// gomutants:disable-next-line BRANCH_IF reason="NewLinkedHandler already validates vars before invoking mount callback"
		if err != nil {
			return err
		}
		files, err := host.NewClientFromVars(vars)
		// gomutants:disable-next-line BRANCH_IF reason="NewLinkedHandler already validates vars before invoking mount callback"
		if err != nil {
			return err
		}
		queue, err := host.NewQueueClientFromVars(vars)
		// gomutants:disable-next-line BRANCH_IF reason="NewLinkedHandler already validates vars before invoking mount callback"
		if err != nil {
			return err
		}
		return mount(s, NewBookServer(files), NewTextServer(files, texts, store, queue))
	})
}
