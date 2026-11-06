// Package service is the backend of the OCR plugin: the text provider that
// recognises the text of a file on host request, and the commands that recognise a file.
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

// newEngineOpener returns what opens the models of the configuration through the library, or the
// mock where the host runs on mock models.
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

// updateConfig applies the settings received from the host. A resolution that is
// not a positive number is ignored, as is a model the plugin does not provision.
// The paths of a custom model are received only while the custom model is
// selected, and are empty in every other case; a custom model without its file
// reads with the model the manifest provisions for the custom option.
func (e *engines) updateConfig(values map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if dpi, err := strconv.Atoi(values["dpi"]); err == nil && dpi > 0 && dpi != e.cfg.Recognise.DPI {
		e.cfg.Recognise.DPI = dpi
		e.dropLocked()
	}
	recognise := e.cfg.Recognise
	model := values["model"]
	if model == engine.RecogniserCustom {
		model = engine.RecogniserCustomFallback
	}
	if files, ok := engine.FindRecognisers(model); ok {
		recognise.Files = files
	}
	recognise.Path, recognise.Dict = values["customModelPath"], values["customDictPath"]
	if !recognise.IsEqual(e.cfg.Recognise) {
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
	if err := commands.Add("recognition.create", newRecogniseCommand(recognitions)); err != nil {
		return err
	}
	if err := commands.Add("recognition.replace", newReplaceTextLayerCommand(recognitions)); err != nil {
		return err
	}
	if err := commands.Add("recognition.delete", command.NewDeleteTextRun(h.Texts, h.Store, PluginID)); err != nil {
		return err
	}
	s.SetUpdateConfig(models.updateConfig)
	s.Register(host.NewTextProviderHandler(provider, s.HandlerOptions()...))
	command.Mount(s, commands)
	s.OnClose(func() error {
		provider.Close()
		return models.close()
	})
	return nil
}

// replaceTextLayerTitle is the title the manifest gives recognition.replace.
const replaceTextLayerTitle = "Recognise text over the text layer"

// newRecogniseCommand recognises the file named by the user and waits for completion. A file with a
// text layer keeps it.
func newRecogniseCommand(recognitions *recognition.Recognition) command.Run {
	return newRecognitionCommand(recognitions, false)
}

// newReplaceTextLayerCommand recognises the file named by the user over its text layer and waits
// for completion.
func newReplaceTextLayerCommand(recognitions *recognition.Recognition) command.Run {
	return newRecognitionCommand(recognitions, true)
}

func newRecognitionCommand(recognitions *recognition.Recognition, shouldReplaceTextLayer bool) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		res, err := recognitions.Recognise(ctx, recognition.Request{
			Vault: "", Path: req.Path, IsUserStarted: true, ShouldReplaceTextLayer: shouldReplaceTextLayer,
		})
		if err != nil {
			return command.Result{}, err
		}
		switch {
		case res.HasTextLayer:
			return command.Result{
				Message: fmt.Sprintf("%s has a text layer, which stays the text of its %d pages. Run %q to replace it with a recognition",
					req.Path, res.Pages, replaceTextLayerTitle),
				OpenedPath: "",
			}, nil
		case res.IsEmpty:
			return command.Result{
				Message:    fmt.Sprintf("%s says nothing that could be read, and nothing was kept", req.Path),
				OpenedPath: "",
			}, nil
		default:
			return command.Result{
				Message:    fmt.Sprintf("recognised %d pages of %s", res.Pages, req.Path),
				OpenedPath: "",
			}, nil
		}
	}
}
