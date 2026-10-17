package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

var (
	errRecognitionUnavailable = errors.New("recognition plugin is unavailable or not installed")
)

// RecogniserReady says whether a document could be read now without waiting for
// anything to arrive.
func (c Config) RecogniserReady() bool {
	if c.PluginHost != nil && !c.SafeMode {
		if _, ok := c.PluginHost.FindPluginByCapability("recognition"); ok {
			return true
		}
	}
	return false
}

// PrepareRecogniser makes the runtime this process reads a page through.
func (c Config) PrepareRecogniser(_ context.Context) error {
	return nil
}

// Recogniser is what reads a scanned page on this machine, opened now.
func (c Config) Recogniser(ctx context.Context) (recogniser port.Recogniser, closer func() error, why error) {
	if host, ok := c.findPluginHost("recognition"); ok {
		bridge, err := plugin.OpenRecogniserBridge(ctx, host)
		if err != nil {
			return nil, nil, err
		}
		return bridge, bridge.Close, nil
	}
	return nil, nil, errRecognitionUnavailable
}

// Recognise is one document read with the recogniser given.
func (c Config) Recognise(sources port.SourceRepository, by port.Recogniser) recognition.Recognise {
	return recognition.NewRecognise(c.VaultReaders(), sources, c.GetDerivedStores(), c.PageRenderer(), by)
}

// OpenRecognitionWorker is the queue that reads this installation's scanned documents.
func (c Config) OpenRecognitionWorker(
	ctx context.Context,
	sources port.SourceRepository,
	tasks *run.Tasks,
	models *run.Lock,
) *recognition.Worker {
	return recognition.NewWorker(ctx, recognition.Config{
		Readers:   c.VaultReaders(),
		Derived:   c.GetDerivedStores(),
		Documents: c.PageRenderer(),
		Sources:   sources,
		Tasks:     tasks,
		Models:    models,
		Runtime: recognition.Runtime{
			Open:     c.openRecognitionRuntime,
			Ready:    c.RecogniserReady,
			Prepared: c.isRecognitionPrepared,
		},
		Proofreading: c.getProofreading(c.ScanProofreading),
	})
}

func (c Config) openRecognitionRuntime(
	ctx context.Context, _ func(what string, done, total int64),
) (port.Recogniser, func() error, error) {
	if host, ok := c.findPluginHost("recognition"); ok {
		bridge, err := plugin.OpenRecogniserBridge(ctx, host)
		if err != nil {
			return nil, nil, err
		}
		return bridge, bridge.Close, nil
	}
	return nil, nil, errRecognitionUnavailable
}

func (c Config) isRecognitionPrepared() bool {
	if c.PluginHost != nil && !c.SafeMode {
		if _, ok := c.PluginHost.FindPluginByCapability("recognition"); ok {
			return true
		}
	}
	return false
}
