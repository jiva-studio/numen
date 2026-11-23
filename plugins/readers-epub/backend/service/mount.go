package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/internal/adapter/hostclient"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/books"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/texts"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/wire/bookwire"
)

// ShownVault returns the vault the host shows, whose feed the plugin follows when it starts.
type ShownVault func(ctx context.Context) (domain.Vault, error)

// App is the backend of the plugin: Mount registers its services over the host the environment names,
// and HandleEvent follows the file change feed of the vault an onFilesChanged event names.
type App struct {
	follower *texts.Follower
}

func NewApp() *App { return &App{follower: nil} }

// Mount registers the plugin's services on s and follows the feed of the shown vault.
func (a *App) Mount(s *server.Server) error {
	connected, err := hostclient.OpenFromEnv()
	if err != nil {
		return err
	}
	a.follower, err = mount(s, connected)
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

func mount(s *server.Server, connected *hostclient.Host) (*texts.Follower, error) {
	reader := books.NewReader(connected.OpenFiles)
	methods := bookwire.NewMethods(reader)
	writer := texts.NewWriter(connected.OpenVault, connected.OpenTexts, connected.OpenStore, connected.GetQueue(), connected.ReportError)
	follower := texts.NewFollower(writer, connected.OpenFeed, connected.ReportError)
	// gomutants:disable-next-line STATEMENT_REMOVE reason="server close hook releases cached books"
	s.AddCloser(func() error {
		reader.Close()
		return nil
	})
	// gomutants:disable-next-line STATEMENT_REMOVE reason="server close hook stops running text extraction jobs"
	s.AddCloser(func() error {
		follower.Close()
		writer.Close()
		return nil
	})
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="mount propagates any API registration failure"
	if err := mountMethods(s, methods); err != nil {
		return nil, err
	}
	s.RegisterRoute(entriesPattern, NewEntryServer(methods, connected.ReportError))
	commands, err := newCommands(writer)
	// gomutants:disable-next-line BRANCH_IF reason="registering one static command id on a new registry cannot fail"
	if err != nil {
		return nil, err
	}
	command.Mount(s, commands)
	followShownVault(s, connected.GetShownVault, follower)
	return follower, nil
}

func followShownVault(s *server.Server, shown ShownVault, follower *texts.Follower) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if v, err := shown(ctx); err == nil {
			follower.Wake(ctx, v.ID)
		}
	}()
	s.AddCloser(func() error {
		cancel()
		<-done
		return nil
	})
}
