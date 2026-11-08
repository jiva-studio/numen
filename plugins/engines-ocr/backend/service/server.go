// Package service is the backend of the OCR plugin: the text provider that
// recognises the text of a file on host request, the recognise command, and the agent tool
// recognition_create.
package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/onnx"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

// EnvMockML names the variable that has the plugin read every page with the mock recogniser.
const EnvMockML = "NUMEN_MOCK_ML"

// engineFactory opens the recogniser the configuration names.
type engineFactory func(ctx context.Context, cfg engine.Config) (recogniser, error)

// recogniser is the recogniser of a page and what it holds.
type recogniser interface {
	recognition.Recogniser
	Close() error
}

func newEngineOpener(library *onnx.Library) engineFactory {
	return func(ctx context.Context, cfg engine.Config) (recogniser, error) {
		if os.Getenv(EnvMockML) == "true" {
			return engine.MockRecogniser{}, nil
		}
		return engine.Open(ctx, library, cfg)
	}
}

// engines holds the recogniser, opened when a page is first read and opened again once a setting
// that decides what it reads with changes.
type engines struct {
	mu      sync.Mutex
	cfg     engine.Config
	factory engineFactory
	engine  recogniser
}

func newEngines(cfg engine.Config, factory engineFactory) *engines {
	return &engines{mu: sync.Mutex{}, cfg: cfg, factory: factory, engine: nil}
}

func (e *engines) updateConfig(values map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if dpi, err := strconv.Atoi(values["dpi"]); err == nil && dpi > 0 && dpi != e.cfg.Recognise.DPI {
		e.cfg.Recognise.DPI = dpi
		e.dropLocked()
	}
	recognise := e.cfg.Recognise
	if address, ok := engine.FindRecogniser(values["model"]); ok {
		recognise.Name = address
		if address == "" {
			recognise.Name = engine.RecogniserCustom
		}
	}
	recognise.Path, recognise.Dict = values["customModelPath"], values["customDictPath"]
	if recognise != e.cfg.Recognise {
		e.cfg.Recognise = recognise
		e.dropLocked()
	}
}

func (e *engines) getEngine(ctx context.Context) (recognition.Recogniser, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.engine != nil {
		return e.engine, nil
	}
	opened, err := e.factory(context.WithoutCancel(ctx), e.cfg)
	if err != nil {
		return nil, err
	}
	e.engine = opened
	return opened, nil
}

func (e *engines) close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.closeLocked()
}

func (e *engines) dropLocked() {
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="Stderr logging of close failure does not affect engine state."
	if err := e.closeLocked(); err != nil {
		fmt.Fprintln(os.Stderr, "engines-ocr: close the recogniser:", err)
	}
}

func (e *engines) closeLocked() error {
	if e.engine == nil {
		return nil
	}
	err := e.engine.Close()
	e.engine = nil
	return err
}

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
	return mount(s, recognition.Host{Vault: vault, Store: store, Texts: texts, Queue: queue}, newEngines(getConfig(), newEngineOpener(engine.NewLibrary())))
}

func getConfig() engine.Config {
	cfg := engine.Defaults()
	if dir := os.Getenv("NUMEN_OCR_DIR"); dir != "" {
		cfg.Dir = dir
	} else if modelsDir := os.Getenv(engine.EnvDir); modelsDir != "" {
		cfg.Dir = modelsDir
	}
	if runtime := os.Getenv("NUMEN_OCR_RUNTIME"); runtime != "" {
		cfg.Runtime = runtime
	}
	return cfg
}

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.engines.ocr"

func mount(s *server.Server, h recognition.Host, models *engines) error {
	recognitions, err := recognition.New(h, models.getEngine)
	if err != nil {
		return err
	}
	agentTools := NewTools(recognitions)
	commands, err := newCommands(agentTools)
	if err != nil {
		return err
	}
	registry, err := agentTools.Registry()
	if err != nil {
		return err
	}
	s.SetUpdateConfig(models.updateConfig)
	command.Mount(s, commands)
	registry.Mount(s)
	s.OnClose(func() error {
		agentTools.Close()
		return models.close()
	})
	return nil
}

func newCommands(agentTools *Tools) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add("app:file.createText", newRecogniseCommand(agentTools)); err != nil {
		return nil, err
	}
	return commands, nil
}

func newRecogniseCommand(agentTools *Tools) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		if _, err := agentTools.recognitions.StatFile(ctx, "", req.Path); err != nil {
			return command.Result{}, err
		}
		state, err := agentTools.start(ctx, "", req.Path)
		if err != nil {
			return command.Result{}, err
		}
		if state == stateRunning {
			return command.Result{}, fmt.Errorf("%s: %w", req.Path, recognition.ErrAlreadyRecognising)
		}
		return command.Result{Message: "recognising " + req.Path, OpenedPath: ""}, nil
	}
}
