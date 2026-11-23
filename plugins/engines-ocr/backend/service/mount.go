// Package service is the backend of the OCR plugin: the text provider that recognizes the text of a
// file on host request, the recognize command, the recognition of the scans the file change feed names
// on each onFilesChanged event, and the agent tool recognition_create.
package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/document"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/engine"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/engines"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/feed"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.engines.ocr"

// App is the backend of the plugin: Mount registers its services, and HandleEvent recognizes the scans
// the file change feed names on each onFilesChanged event while Recognize automatically is on.
type App struct {
	auto *feed.Automatic
}

func NewApp() *App { return &App{auto: nil} }

// HandleEvent recognizes the scans the feed of the vault an onFilesChanged event names.
func (a *App) HandleEvent(ctx context.Context, event string) error {
	if a.auto == nil {
		return nil
	}
	return a.auto.HandleEvent(ctx, event)
}

// Mount registers the services of the plugin on s, working the vault, the plugin's folder, the text
// versions and the queue through the host the environment names. The recognizer closes with s.
func (a *App) Mount(s *server.Server) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	vault := host.NewClient(endpoint)
	store := host.NewStoreClient(endpoint)
	texts := host.NewTextClient(endpoint)
	queue := host.NewQueueClient(endpoint)
	fetch := host.NewDependencyClient(endpoint).FetchModelFile
	env := process.Process{}
	holder := engines.NewHolder(env.GetConfig(), newOpener(engine.NewLibrary(), env.IsMockML, fetch), env)
	feeds := func(id domain.VaultID) feed.Vault { return feed.Vault{Feed: vault.With(id), Store: store.With(id)} }
	auto, err := mount(s, recognition.Host{
		GetVault:    func(id domain.VaultID) recognition.Files { return vault.With(id) },
		Store:     func(id domain.VaultID) recognition.Folder { return store.With(id) },
		GetTexts:     func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		Queue:     queue,
		Documents: document.NewCachedReader(),
		Report:    env.ReportError,
	}, feeds, holder)
	a.auto = auto
	return err
}

func mount(s *server.Server, h recognition.Host, feeds func(domain.VaultID) feed.Vault, holder *engines.Holder) (*feed.Automatic, error) {
	recognitions, err := recognition.New(h, holder.GetEngine)
	if err != nil {
		return nil, err
	}
	launcher := recognition.NewLauncher(recognitions, PluginID)
	commands, err := newCommands(launcher)
	if err != nil {
		return nil, err
	}
	registry, err := newToolRegistry(launcher)
	if err != nil {
		return nil, err
	}
	auto := feed.NewAutomatic(newFollower(launcher, feeds, h.Report), launcher.GetShownVaultID, getReport(h.Report))
	s.SetUpdateConfig(func(values map[string]string) {
		holder.UpdateSettings(values)
		auto.UpdateSettings(values)
	})
	command.Mount(s, commands)
	registry.Mount(s)
	s.OnClose(func() error {
		auto.Close()
		launcher.Close()
		return holder.Close()
	})
	return auto, nil
}

func newFollower(launcher *recognition.Launcher, feeds func(domain.VaultID) feed.Vault, report func(string, error)) *feed.Follower {
	return feed.New(feeds, launchRuns{launcher: launcher}, getReport(report))
}

func getReport(report func(string, error)) func(string, error) {
	if report == nil {
		return func(string, error) {}
	}
	return report
}

func newCommands(launcher *recognition.Launcher) (*command.Registry, error) {
	commands := command.New()
	return commands, commands.Add(command.CreateText, newRecognizeCommand(launcher))
}

func newRecognizeCommand(launcher *recognition.Launcher) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		if err := launcher.StartFile(ctx, req.Path); err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: "recognizing " + req.Path, OpenedPath: ""}, nil
	}
}

// launchRuns starts the recognitions the follower asks for through the launcher.
type launchRuns struct {
	launcher *recognition.Launcher
}

func (r launchRuns) Start(ctx context.Context, vault domain.VaultID, path string) (feed.State, error) {
	out, err := r.launcher.CreateInBackground(ctx, vault, path)
	if err != nil {
		return 0, err
	}
	switch out.State {
	case recognition.StateDone:
		return feed.StateDone, nil
	case recognition.StateRunning:
		return feed.StateRunning, nil
	}
	return feed.StateStarted, nil
}

func (r launchRuns) IsRunning(ctx context.Context, vault domain.VaultID, path string) (bool, error) {
	return r.launcher.IsRunning(ctx, vault, path)
}
