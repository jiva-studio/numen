// Package service is the backend of the transcription plugin: it transcribes the recordings a user
// requests, writes the transcripts into the plugin's own folder of the vault's store in the common
// format of text versions, and records each with the host as a version of kind "transcript".
package service

import (
	"context"
	"fmt"
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

// GetActiveVaultID returns the id of the vault the host shows.
func (f vaultFiles) GetActiveVaultID(ctx context.Context) (domain.VaultID, error) {
	shown, err := f.vault.GetVault(ctx)
	if err != nil {
		return "", err
	}
	return shown.ID, nil
}

// openModel opens the recogniser of the configuration.
type openModel func(ctx context.Context, cfg engine.Config) (engine.TranscriberPort, error)

func newParakeetOpener(runtime engine.Runtime) openModel {
	return func(ctx context.Context, cfg engine.Config) (engine.TranscriberPort, error) {
		return engine.Open(ctx, runtime, cfg)
	}
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

func (m *models) updateConfig(values map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if model, ok := engine.FindModel(values["model"]); ok {
		m.cfg.Model = model
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

// Mount registers the handlers and the agent tool transcript_create on s over the host given. The model
// is the mock one where isMock is set. The transcriptions and the model end with s.
func Mount(s *server.Server, hosted Host, cfg engine.Config, isMock bool) error {
	model := &models{mu: sync.Mutex{}, cfg: cfg, isMock: isMock, open: newParakeetOpener(engine.NewRuntime()), current: nil}
	transcriber := NewTranscriber(vaultFiles{vault: hosted.Vault}, hosted.Store, hosted.Texts, hosted.Queue, model.get)
	s.SetUpdateConfig(func(values map[string]string, _ server.Secrets) { model.updateConfig(values) })
	commands, err := newCommands(transcriber)
	if err != nil {
		return err
	}
	registry, err := NewTools(transcriber, hosted.Vault).Registry()
	if err != nil {
		return err
	}
	command.Mount(s, commands)
	registry.Mount(s)
	s.OnClose(func() error {
		transcriber.Close()
		return model.close()
	})
	return nil
}

func newCommands(transcriber *Transcriber) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add(command.CreateText, newTranscribeCommand(transcriber)); err != nil {
		return nil, err
	}
	return commands, nil
}

func newTranscribeCommand(transcriber *Transcriber) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		isStarted, err := transcriber.Start(ctx, "", req.Path)
		if err != nil {
			return command.Result{}, err
		}
		if !isStarted {
			return command.Result{}, fmt.Errorf("%s: %w", req.Path, errAlreadyTranscribing)
		}
		return command.Result{Message: "transcribing " + req.Path, OpenedPath: ""}, nil
	}
}

// MountFromEnv mounts the plugin over the host the process was started for.
func MountFromEnv(s *server.Server, cfg engine.Config) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	hosted := openHostAt(endpoint)
	return Mount(s, hosted, cfg, os.Getenv(EnvMockML) == "true")
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
		return Mount(s, hosted, NewConfig("", ""), vars[EnvMockML] == "true")
	})
}
