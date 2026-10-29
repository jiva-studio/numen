// Package service is the backend of the transcription plugin: it transcribes the recordings a user
// requests, writes the transcripts into the plugin's own folder of the vault's store in the common
// format of text versions, and records each with the host as a version of kind "transcript".
package service

import (
	"context"
	"net/http"
	"os"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.engines.transcription"

// EnvMockML names the variable that has the plugin hear every recording with the mock model.
const EnvMockML = "NUMEN_MOCK_ML"

// Host is what the plugin works through the host.
type Host struct {
	Vault host.Vault
	Store host.Store
	Texts host.Texts
	Queue host.Queue
}

// openHostAt connects to the host at the endpoint.
func openHostAt(endpoint hostcall.Endpoint) Host {
	return Host{
		Vault: host.NewClient(endpoint),
		Store: host.NewStoreClient(endpoint),
		Texts: host.NewTextClient(endpoint),
		Queue: host.NewQueueClient(endpoint),
	}
}

// vaultFiles reads files through the host.
type vaultFiles struct {
	vault host.Vault
}

func (f vaultFiles) ReadFile(ctx context.Context, vault domain.VaultID, path string) ([]byte, error) {
	reader := f.vault
	if vault != "" {
		reader = reader.With(vault)
	}
	raw, _, err := reader.Read(ctx, path)
	return raw, err
}

// openModel opens the recogniser of the configuration.
type openModel func(ctx context.Context, cfg engine.Config) (engine.TranscriberPort, error)

func openParakeet(ctx context.Context, cfg engine.Config) (engine.TranscriberPort, error) {
	return engine.Open(ctx, cfg)
}

// models opens the model once, under the configuration the settings last named, and keeps it until
// the plugin stops.
type models struct {
	mu      sync.Mutex
	cfg     engine.Config
	isMock  bool
	open    openModel
	current engine.TranscriberPort
}

func (m *models) get(ctx context.Context) (engine.TranscriberPort, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current != nil {
		return m.current, nil
	}
	if m.isMock {
		m.current = engine.NewMockTranscriber()
		return m.current, nil
	}
	loaded, err := m.open(context.WithoutCancel(ctx), m.cfg)
	if err != nil {
		return nil, err
	}
	m.current = loaded
	return m.current, nil
}

// updateConfig takes the model and the language the settings name, which the model opens with. A
// model the plugin does not provision is left as it was. The host stops the plugin when the model
// changes, since the model's files change with it, so an open model is kept.
func (m *models) updateConfig(values map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if model, ok := engine.FindModel(values["model"]); ok {
		m.cfg.Model = model
	}
	if language, ok := values["language"]; ok {
		m.cfg.Language = language
	}
}

func (m *models) close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return nil
	}
	err := m.current.Close()
	m.current = nil
	return err
}

// NewConfig is the engine configuration the environment gives, with the directory of the models and the
// ONNX runtime library a command line names laid over it.
func NewConfig(modelsDir, runtime string) engine.Config {
	cfg := engine.Defaults()
	if modelsDir != "" {
		cfg.Dir = modelsDir
	} else if envDir := os.Getenv(engine.EnvDir); envDir != "" {
		cfg.Dir = envDir
	}
	if runtime != "" {
		cfg.Runtime = runtime
	} else if envRuntime := os.Getenv("NUMEN_ONNX_RUNTIME"); envRuntime != "" {
		cfg.Runtime = envRuntime
	}
	return cfg
}

// Mount registers the text provider on s over the host given. The model
// is the mock one where isMock is set. The transcriptions and the model end with s.
func Mount(s *server.Server, hosted Host, cfg engine.Config, isMock bool) {
	model := &models{cfg: cfg, isMock: isMock, open: openParakeet}
	transcriber := NewTranscriber(vaultFiles{vault: hosted.Vault}, hosted.Store, hosted.Texts, hosted.Queue, model.get)
	s.SetUpdateConfig(model.updateConfig)
	s.Register(host.NewTextProviderHandler(provider{transcriber: transcriber}, s.HandlerOptions()...))
	commands := command.New()
	// The registry is new, so the one id cannot be added twice.
	_ = commands.Add("transcribe", newTranscribeCommand(transcriber))
	command.Mount(s, commands)
	s.OnClose(func() {
		transcriber.Close()
		// The plugin is stopping, and a model that will not close leaves nothing to retry.
		_ = model.close()
	})
}

// newTranscribeCommand transcribes the recording the user named, in the vault the host has open, and
// waits for it.
func newTranscribeCommand(transcriber *Transcriber) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		if err := transcriber.Transcribe(ctx, "", req.Path); err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: "transcribed " + req.Path}, nil
	}
}

// MountFromEnv mounts the plugin over the host the process was started for.
func MountFromEnv(s *server.Server, cfg engine.Config) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	hosted := openHostAt(endpoint)
	Mount(s, hosted, cfg, os.Getenv(EnvMockML) == "true")
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	endpoint, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	hosted := openHostAt(endpoint)
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		Mount(s, hosted, NewConfig("", ""), vars[EnvMockML] == "true")
		return nil
	})
}
