// Package service is the backend of the OCR plugin: the text provider that recognizes the text of a
// file on host request, the recognize command, the command that recognizes the scans the file change
// feed names, and the agent tool recognition_create.
package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/document"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/engine"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/engines"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/feed"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/launch"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.engines.ocr"

// UpdateCommand is the command that recognizes the scans of the shown vault that its file change feed
// names as added or changed. A trigger on file.added and file.changed runs it while Recognize
// automatically is on; the scans are read from the feed, not from the trigger.
const UpdateCommand = "recognitions.update"

// Mount registers the services of the plugin on s, working the vault, the plugin's folder, the text
// versions and the queue through the host the environment names. The recognizer closes with s.
func Mount(s *server.Server) error {
	vault, err := host.NewClientFromEnv()
	if err != nil {
		return err
	}
	store, err := host.NewStoreClientFromEnv()
	if err != nil {
		return err
	}
	texts, err := host.NewTextClientFromEnv()
	if err != nil {
		return err
	}
	queue, err := host.NewQueueClientFromEnv()
	if err != nil {
		return err
	}
	env := process.Process{}
	holder := engines.NewHolder(env.GetConfig(), newOpener(engine.NewLibrary(), env.IsMockML), env)
	feeds := func(id domain.VaultID) feed.Vault { return feed.Vault{Feed: vault.With(id), Store: store.With(id)} }
	return mount(s, recognition.Host{
		Vaults:    func(id domain.VaultID) recognition.Files { return vault.With(id) },
		Store:     func(id domain.VaultID) recognition.Folder { return store.With(id) },
		Texts:     func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		Queue:     queue,
		Documents: document.NewCachedReader(),
		Report:    env.ReportError,
	}, feeds, holder)
}

func mount(s *server.Server, h recognition.Host, feeds func(domain.VaultID) feed.Vault, holder *engines.Holder) error {
	recognitions, err := recognition.New(h, holder.GetEngine)
	if err != nil {
		return err
	}
	launcher := launch.New(recognitions, PluginID)
	commands, err := newCommands(launcher, newFollower(launcher, feeds, h.Report))
	if err != nil {
		return err
	}
	registry, err := newToolRegistry(launcher)
	if err != nil {
		return err
	}
	s.SetUpdateConfig(func(values map[string]string) { holder.UpdateSettings(values) })
	command.Mount(s, commands)
	registry.Mount(s)
	s.OnClose(func() error {
		launcher.Close()
		return holder.Close()
	})
	return nil
}

func newFollower(launcher *launch.Launcher, feeds func(domain.VaultID) feed.Vault, report func(string, error)) *feed.Follower {
	if report == nil {
		report = func(string, error) {}
	}
	return feed.New(feeds, launchRuns{launcher: launcher}, report)
}

func newCommands(launcher *launch.Launcher, follower *feed.Follower) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add(command.CreateText, newRecognizeCommand(launcher)); err != nil {
		return nil, err
	}
	return commands, commands.Add(UpdateCommand, newUpdateCommand(launcher, follower))
}

func newRecognizeCommand(launcher *launch.Launcher) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		if err := launcher.StartFile(ctx, req.Path); err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: "recognizing " + req.Path, OpenedPath: ""}, nil
	}
}

func newUpdateCommand(launcher *launch.Launcher, follower *feed.Follower) command.Run {
	return func(ctx context.Context, _ command.Request) (command.Result, error) {
		vault, err := launcher.GetShownVaultID(ctx)
		if err != nil {
			return command.Result{}, err
		}
		if err := follower.Execute(ctx, vault); err != nil {
			return command.Result{}, err
		}
		return command.Result{}, nil
	}
}

// launchRuns starts the recognitions the follower asks for through the launcher.
type launchRuns struct {
	launcher *launch.Launcher
}

func (r launchRuns) Start(ctx context.Context, vault domain.VaultID, path string) (feed.State, error) {
	out, err := r.launcher.Create(ctx, vault, path)
	if err != nil {
		return 0, err
	}
	switch out.State {
	case launch.StateDone:
		return feed.StateDone, nil
	case launch.StateRunning:
		return feed.StateRunning, nil
	}
	return feed.StateStarted, nil
}

func (r launchRuns) IsRunning(ctx context.Context, vault domain.VaultID, path string) (bool, error) {
	return r.launcher.IsRunning(ctx, vault, path)
}
