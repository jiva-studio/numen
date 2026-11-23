package service

import (
	"context"

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

// ShownVault names the vault the host shows, whose feed the plugin follows when it starts.
type ShownVault interface {
	GetVault(ctx context.Context) (domain.Vault, error)
}

// App is the backend of the plugin: Mount registers its services over the host the environment names,
// and HandleEvent follows the file change feed of the vault an onFilesChanged event names.
type App struct {
	follower *extract.Follower
}

func NewApp() *App { return &App{follower: nil} }

// Mount registers the plugin's services on s and follows the feed of the shown vault.
func (a *App) Mount(s *server.Server) error {
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
	a.follower, err = mountClients(s, files, texts, store, queue)
	return err
}

// HandleEvent follows the feed of the vault an onFilesChanged event names, in the background, and
// ignores any other event.
func (a *App) HandleEvent(ctx context.Context, event string) error {
	vault, ok := host.ReadFilesChangedEvent(event)
	if ok && a.follower != nil {
		a.follower.Wake(ctx, vault)
	}
	return nil
}

func mountClients(
	s *server.Server, files *host.Client, texts *host.TextClient, store *host.StoreClient, queue host.Queue,
) (*extract.Follower, error) {
	workers := pdfworkers.New(clock.StartTimer)
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
	if err := mount(s, reader, writer, follower, pdfworkers.ReportError); err != nil {
		return nil, err
	}
	followShownVault(s, files, follower)
	return follower, nil
}

func mount(
	s *server.Server, reader *document.Reader, writer *extract.Writer, follower *extract.Follower, report func(what string, err error),
) error {
	if err := NewDocumentServer(reader).Mount(s.NewAPI(api.Service)); err != nil {
		return err
	}
	s.RegisterRoute(pagesPattern, NewPageServer(reader, report))
	commands, err := newCommands(writer)
	if err != nil {
		return err
	}
	command.Mount(s, commands)
	s.AddCloser(func() error {
		follower.Close()
		writer.Close()
		return nil
	})
	return nil
}

func followShownVault(s *server.Server, shown ShownVault, follower *extract.Follower) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if v, err := shown.GetVault(ctx); err == nil {
			follower.Wake(ctx, v.ID)
		}
	}()
	s.AddCloser(func() error {
		cancel()
		<-done
		return nil
	})
}
