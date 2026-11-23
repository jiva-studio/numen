// Package service is the backend of the transcription plugin: it builds the adapters and the use
// cases and registers the commands, the settings and the agent tool with the host, and transcribes the
// recordings the file change feed names on each onFilesChanged event.
package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/engine"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/hostclient"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/feed"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/model"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/transcribe"
)

// App is the backend of the plugin: Mount registers its services, and HandleEvent transcribes the
// recordings the file change feed names on each onFilesChanged event while Transcribe automatically is
// on.
type App struct {
	auto *feed.Automatic
}

func NewApp() *App { return &App{auto: nil} }

// HandleEvent transcribes the recordings the feed of the vault an onFilesChanged event names.
func (a *App) HandleEvent(ctx context.Context, event string) error {
	if a.auto == nil {
		return nil
	}
	return a.auto.HandleEvent(ctx, event)
}

// Mount registers the handlers and the agent tool transcript_create on s over the host the process
// was started for. The model is the mock one where NUMEN_TEST_MOCK_ML is "true". The transcriptions and
// the model end with s.
func (a *App) Mount(s *server.Server) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	env := process.Process{}
	a.auto, err = mount(s, hostclient.Open(endpoint), env.GetConfig(), env.IsMockML())
	return err
}

func mount(s *server.Server, hosted hostclient.Host, cfg settings.Config, isMock bool) (*feed.Automatic, error) {
	holder := model.NewHolder(cfg, newOpener(engine.NewRuntime(), isMock, hosted.FetchModelFile))
	report := process.Process{}.ReportError
	transcriber := transcribe.NewTranscriber(hostclient.NewFiles(hosted.GetVault), hosted.Store, hosted.GetTexts, hosted.Queue, holder.GetRecognizer, report)
	requests := transcribe.NewRequests(transcriber, func(id domain.VaultID) transcribe.Vault { return hosted.GetVault(id) })
	feeds := func(id domain.VaultID) feed.Vault {
		return feed.Vault{Feed: hosted.GetVault(id), Store: hosted.Store(id)}
	}
	auto := feed.NewAutomatic(newFollower(requests, feeds, report), requests.GetShownVaultID, report)
	s.SetUpdateConfig(func(values map[string]string) {
		holder.UpdateSettings(values)
		auto.UpdateSettings(values)
	})
	commands, err := newCommands(transcriber)
	if err != nil {
		return nil, err
	}
	registry, err := newTools(requests).buildRegistry()
	if err != nil {
		return nil, err
	}
	command.Mount(s, commands)
	registry.Mount(s)
	s.OnClose(func() error {
		auto.Close()
		transcriber.Close()
		return holder.Close()
	})
	return auto, nil
}
