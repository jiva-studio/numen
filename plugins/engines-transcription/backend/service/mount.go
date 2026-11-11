// Package service is the backend of the transcription plugin: it builds the adapters and the use
// cases and registers the commands, the settings and the agent tool with the host.
package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/engine"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/hostclient"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/model"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/transcribe"
)

// Mount registers the handlers and the agent tool transcript_create on s over the host the process
// was started for. The model is the mock one where NUMEN_MOCK_ML is "true". The transcriptions and
// the model end with s.
func Mount(s *server.Server) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	env := process.Process{}
	return mount(s, hostclient.Open(endpoint), env.GetConfig(), env.IsMockML())
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	endpoint, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	hosted := hostclient.Open(endpoint)
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return mount(s, hosted, process.Process{}.GetConfig(), process.IsMockML(vars))
	})
}

func mount(s *server.Server, hosted hostclient.Host, cfg settings.Config, isMock bool) error {
	holder := model.NewHolder(cfg, newOpener(engine.NewRuntime(), isMock))
	transcriber := transcribe.NewTranscriber(hostclient.NewFiles(hosted.Vaults), hosted.Store, hosted.Texts, hosted.Queue, holder.GetRecognizer)
	transcriber.SetReport(process.Process{}.ReportError)
	requests := transcribe.NewRequests(transcriber, func(id domain.VaultID) transcribe.Vault { return hosted.Vaults(id) })
	s.SetUpdateConfig(func(values map[string]string, _ server.Secrets) { holder.UpdateSettings(values) })
	commands, err := newCommands(transcriber)
	if err != nil {
		return err
	}
	registry, err := newTools(requests).registry()
	if err != nil {
		return err
	}
	command.Mount(s, commands)
	registry.Mount(s)
	s.OnClose(func() error {
		transcriber.Close()
		return holder.Close()
	})
	return nil
}
