// Package service is the backend of the transcription plugin: it hears the recordings a person asks
// for, keeps their transcripts in the plugin's own folder of the vault's store, gives them to the
// host as timed text versions, and serves TranscriptService to the player.
package service

import (
	"context"
	"net/http"
	"os"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
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

// openHostAt reaches the host at the endpoint.
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
	raw, _, err := f.vault.With(vault).Read(ctx, path)
	return raw, err
}

// models opens the model once, under the configuration the settings last named, and keeps it until
// the plugin stops.
type models struct {
	mu      sync.Mutex
	cfg     engine.Config
	isMock  bool
	current engine.TranscriberPort
}

func (m *models) open(ctx context.Context) (engine.TranscriberPort, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current != nil {
		return m.current, nil
	}
	if m.isMock {
		m.current = engine.NewMockTranscriber()
		return m.current, nil
	}
	opened, err := engine.Open(context.WithoutCancel(ctx), m.cfg)
	if err != nil {
		return nil, err
	}
	m.current = opened
	return m.current, nil
}

func (m *models) updateConfig(values map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if lang, ok := values["language"]; ok && lang != "" {
		m.cfg.Model.Name = lang
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

// Mount registers the text provider and the transcript service on s over the host given. The model
// is the mock one where isMock is set. The transcriptions and the model end with s.
func Mount(s *server.Server, hosted Host, cfg engine.Config, isMock bool) {
	opened := &models{cfg: cfg, isMock: isMock}
	transcriber := NewTranscriber(vaultFiles{vault: hosted.Vault}, hosted.Store, hosted.Texts, hosted.Queue, opened.open)
	s.SetUpdateConfig(opened.updateConfig)
	s.Register(host.NewTextProviderHandler(provider{stores: hosted.Store, transcriber: transcriber}, s.HandlerOptions()...))
	s.Register(numenv1connect.NewTranscriptServiceHandler(transcriptServer{stores: hosted.Store, texts: hosted.Texts}, s.HandlerOptions()...))
	s.OnClose(func() {
		transcriber.Close()
		// The plugin is stopping, and a model that will not close leaves nothing to retry.
		_ = opened.close()
	})
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
