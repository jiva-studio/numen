// Package service is the backend of the OCR plugin: the text provider the host asks for the
// recognition of a file, and the recognise command.
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

// engineFactory opens the recogniser the configuration names.
type engineFactory func(ctx context.Context, cfg engine.Config) (recogniser, error)

// recogniser is the recogniser of a page and what it holds.
type recogniser interface {
	recognition.Recogniser
	Close() error
}

// openEngine opens the models of the configuration, or the mock where the host runs on mock models.
func openEngine(ctx context.Context, cfg engine.Config) (recogniser, error) {
	if os.Getenv("NUMEN_MOCK_ML") == "true" {
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
	return &engines{cfg: cfg, factory: factory}
}

// updateConfig takes the settings the host hands over. A resolution that is no positive number is
// left as it was.
func (e *engines) updateConfig(values map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if dpi, err := strconv.Atoi(values["dpi"]); err == nil && dpi > 0 && dpi != e.cfg.Recognise.DPI {
		e.cfg.Recognise.DPI = dpi
		e.dropLocked()
	}
	for key, target := range map[string]*string{
		"language": &e.cfg.Recognise.Name, "custom_model_path": &e.cfg.Recognise.Path, "custom_dict_path": &e.cfg.Recognise.Dict,
	} {
		if value, ok := values[key]; ok && *target != value {
			*target = value
			e.dropLocked()
		}
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

// dropLocked closes the recogniser so the next page opens it again. A recogniser that fails to close
// is told on the standard error, the stream the host keeps of the plugin.
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

func mount(s *server.Server, h recognition.Host, models *engines) error {
	recognitions, err := recognition.New(h, models.getEngine)
	if err != nil {
		return err
	}
	provider := recognition.NewProvider(recognitions, h.Store)
	commands := command.New()
	if err := commands.Add("recognise", newRecogniseCommand(recognitions)); err != nil {
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

// newRecogniseCommand recognises the file a person named and waits for it.
func newRecogniseCommand(recognitions *recognition.Recognition) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		res, err := recognitions.Recognise(ctx, recognition.Request{Path: req.Path, IsUserStarted: true})
		if err != nil {
			return command.Result{}, err
		}
		if res.IsEmpty {
			return command.Result{Message: fmt.Sprintf("%s says nothing that could be read, and nothing was kept", req.Path)}, nil
		}
		return command.Result{Message: fmt.Sprintf("recognised %d pages of %s", res.Pages, req.Path)}, nil
	}
}
