package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
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
	return mountClients(s, files, texts, store, queue)
}

func mountClients(s *server.Server, files *host.Client, texts *host.TextClient, store *host.StoreClient, queue host.Queue) error {
	workers := newWorkers()
	readers := func(id domain.VaultID) host.FileReader { return files.With(id) }
	live := func(id domain.VaultID) LiveTexts { return texts.With(id) }
	vaults := func(id domain.VaultID) vaultFiles { return files.With(id) }
	versions := func(id domain.VaultID) host.TextVersions { return texts.With(id) }
	folders := func(id domain.VaultID) host.StoreWriter { return store.With(id) }
	return mount(s, readers, live, NewTextServer(vaults, versions, folders, queue, workers), workers)
}

func mount(s *server.Server, files func(domain.VaultID) host.FileReader, texts func(domain.VaultID) LiveTexts, provider *TextServer, workers *pdf.Workers) error {
	if err := NewDocumentServer(files, texts, workers).Mount(s.NewAPI(APIService)); err != nil {
		return err
	}
	s.RegisterRoute(pagesPattern, NewPageServer(files, workers))
	commands, err := newCommands(provider)
	if err != nil {
		return err
	}
	command.Mount(s, commands)
	s.OnClose(func() error {
		provider.Close()
		return nil
	})
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
		return mountClients(s, files, texts, store, queue)
	})
}
