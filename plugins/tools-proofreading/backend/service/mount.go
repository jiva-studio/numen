// Package service is the backend of the proofreading plugin: it builds the host clients and the
// model adapter, and registers the proofread commands and the settings with the plugin server.
package service

import (
	"errors"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/internal/adapter/model"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/internal/adapter/settings"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/usecase/proofread"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.tools.proofreading"

func openHostAt(endpoint hostcall.Endpoint) proofread.Host {
	store := host.NewStoreClient(endpoint)
	texts := host.NewTextClient(endpoint)
	return proofread.Host{
		Store: func(id domain.VaultID) host.StoreWriter { return store.With(id) },
		Texts: func(id domain.VaultID) proofread.Texts { return texts.With(id) },
		Queue: host.NewQueueClient(endpoint),
	}
}

// Mount registers the proofread and delete commands on s, using the
// host that the environment names. The model is the mock proofreader when the
// environment enables mock models.
func Mount(s *server.Server) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	client, err := connection.NewClientFromEnv()
	if err != nil {
		return err
	}
	complete := model.NewCompleter(client)
	if model.IsMockSelected() {
		complete = model.WithMock(complete)
	}
	return mount(s, openHostAt(endpoint), complete, nil)
}

// NewHandler returns the handler of the plugin's backend for a host that links
// the plugin in, configured from the variables given to a launched plugin
// process. It uses the host that those variables name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	endpoint, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	client, err := connection.NewClientFromVars(vars)
	if err != nil {
		return nil, err
	}
	complete := model.NewCompleter(client)
	if model.IsMockEnabled(vars[model.EnvMockML]) {
		complete = model.WithMock(complete)
	}
	hosted := openHostAt(endpoint)
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return mount(s, hosted, complete, vars)
	})
}

var errIncompleteHost = errors.New("proofreading needs the store, the texts and the queue of the host")

func mount(s *server.Server, h proofread.Host, complete model.Completer, vars map[string]string) error {
	if h.Store == nil || h.Texts == nil || h.Queue == nil {
		return errIncompleteHost
	}
	current := settings.NewStore()
	if len(vars) > 0 {
		current.Update(vars)
	}
	run := newProofreading(h, complete, current)
	commands, err := newCommands(run)
	if err != nil {
		return err
	}
	s.SetUpdateConfig(current.Update)
	command.Mount(s, commands)
	return nil
}

func newProofreading(h proofread.Host, complete model.Completer, current *settings.Store) *proofread.Proofreading {
	return proofread.New(h, proofread.GetProofreadKinds(), model.NewAsk(complete), current.GetRunConfig)
}

func newCommands(run *proofread.Proofreading) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add("proofreading.create", newProofreadCommand(run)); err != nil {
		return nil, err
	}
	if err := commands.Add(command.CreateText, newProofreadCommand(run)); err != nil {
		return nil, err
	}
	return commands, nil
}
