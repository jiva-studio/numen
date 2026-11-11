// Package service is the backend of the OCR plugin: the text provider that
// recognizes the text of a file on host request, the recognize command, and the agent tool
// recognition_create.
package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engines"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/launch"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.engines.ocr"

// Mount registers the services of the plugin on s, working the vault, the plugin's folder, the text
// versions and the queue through the host the environment names. The recogniser closes with s.
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
	return mount(s, recognition.Host{
		Vaults: func(id domain.VaultID) recognition.Files { return vault.With(id) },
		Store:  func(id domain.VaultID) recognition.Folder { return store.With(id) },
		Texts:  func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		Queue:  queue,
		Report: env.ReportError,
	}, holder)
}

func mount(s *server.Server, h recognition.Host, holder *engines.Holder) error {
	recognitions, err := recognition.New(h, holder.GetEngine)
	if err != nil {
		return err
	}
	launcher := launch.New(recognitions, PluginID)
	commands, err := newCommands(launcher)
	if err != nil {
		return err
	}
	registry, err := newToolRegistry(launcher)
	if err != nil {
		return err
	}
	s.SetUpdateConfig(func(values map[string]string, _ server.Secrets) { holder.UpdateSettings(values) })
	command.Mount(s, commands)
	registry.Mount(s)
	s.OnClose(func() error {
		launcher.Close()
		return holder.Close()
	})
	return nil
}

func newCommands(launcher *launch.Launcher) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add(command.CreateText, newRecogniseCommand(launcher)); err != nil {
		return nil, err
	}
	return commands, nil
}

func newRecogniseCommand(launcher *launch.Launcher) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		if err := launcher.StartFile(ctx, req.Path); err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: "recognizing " + req.Path, OpenedPath: ""}, nil
	}
}
