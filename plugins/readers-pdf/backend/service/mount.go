package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/adapter/pdfworkers"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/document"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/extract"
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
	workers := pdfworkers.New(clock.AfterFunc)
	s.SetUpdateConfig(func(values map[string]string) { pdfworkers.Configure(workers, values) })
	reader := document.NewReader(
		func(id domain.VaultID) host.FileReader { return files.With(id) },
		func(id domain.VaultID) document.LiveTexts { return texts.With(id) },
		workers,
	)
	writer := extract.NewWriter(
		func(id domain.VaultID) extract.Files { return files.With(id) },
		func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		func(id domain.VaultID) host.StoreWriter { return store.With(id) },
		queue, workers, pdfworkers.ReportError,
	)
	follower := extract.NewFollower(writer, func(id domain.VaultID) extract.Feed {
		return extract.Feed{Changes: files.With(id), Store: store.With(id)}
	}, pdfworkers.ReportError)
	return mount(s, reader, writer, follower, files, pdfworkers.ReportError)
}

func mount(
	s *server.Server, reader *document.Reader, writer *extract.Writer, follower *extract.Follower, shown ShownVault, report func(what string, err error),
) error {
	if err := NewDocumentServer(reader).Mount(s.NewAPI(api.Service)); err != nil {
		return err
	}
	s.RegisterRoute(pagesPattern, NewPageServer(reader, report))
	commands, err := newCommands(writer, follower, shown)
	if err != nil {
		return err
	}
	command.Mount(s, commands)
	s.OnClose(func() error {
		follower.Close()
		writer.Close()
		return nil
	})
	return nil
}
