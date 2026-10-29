// Package service is the backend of the OCR plugin: the text provider that
// recognises the text of a file on host request, and the recognise command.
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

// openEngine opens the models of the configuration, or the mock where the host runs on mock models.
func openEngine(ctx context.Context, cfg engine.Config) (recogniser, error) {
	if os.Getenv(EnvMockML) == "true" {
		return engine.MockRecogniser{}, nil
	}
	return engine.Open(ctx, cfg)
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

// updateConfig applies the settings received from the host. A resolution that is
// not a positive number is ignored, as is a model the plugin does not provision.
// The paths of a custom model are received only while the custom model is
// selected, and are empty in every other case.
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
	recognise.Path, recognise.Dict = values["custom_model_path"], values["custom_dict_path"]
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

// dropLocked closes the recogniser so the next page reopens it. A close failure
// is written to standard error, the stream the host keeps for the plugin.
func (e *engines) dropLocked() {
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
	return mount(s, recognition.Host{Vault: vault, Store: store, Texts: texts, Queue: queue}, newEngines(getConfig(), openEngine))
}

// getConfig is the configuration the environment of the process names.
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
	provider := recognition.NewProvider(recognitions)
	commands := command.New()
	if err := commands.Add("recognise", newRecogniseCommand(recognitions)); err != nil {
		return err
	}
	if err := commands.Add("delete-recognition", command.NewDeleteTextRun(h.Texts, h.Store, PluginID)); err != nil {
		return err
	}
	s.SetUpdateConfig(models.updateConfig)
	s.Register(host.NewTextProviderHandler(provider, s.HandlerOptions()...))
	command.Mount(s, commands)
	s.OnClose(func() {
		provider.Close()
		if err := models.close(); err != nil {
			fmt.Fprintln(os.Stderr, "engines-ocr: close the recogniser:", err)
		}
	})
	return nil
}

// newRecogniseCommand recognises the file named by the user and waits for completion.
func newRecogniseCommand(recognitions *recognition.Recognition) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		res, err := recognitions.Recognise(ctx, recognition.Request{Vault: "", Path: req.Path, IsUserStarted: true})
		if err != nil {
			return command.Result{}, err
		}
		if res.IsEmpty {
			return command.Result{
				Message:    fmt.Sprintf("%s says nothing that could be read, and nothing was kept", req.Path),
				OpenedPath: "",
			}, nil
		}
		return command.Result{
			Message:    fmt.Sprintf("recognised %d pages of %s", res.Pages, req.Path),
			OpenedPath: "",
		}, nil
	}
}
