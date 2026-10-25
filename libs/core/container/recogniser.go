package container

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/documentbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/recogniserbridge"
	"github.com/jiva-studio/numen/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/libs/core/usecase/run"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/recognitionport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
)

// RecogniserReady says whether a document could be read now without waiting for
// anything to arrive.
func (c Config) RecogniserReady() bool {
	return c.hasPlugin(bridge.CapabilityRecognition)
}

// Recogniser is what reads a scanned page on this machine, opened now.
func (c Config) Recogniser(ctx context.Context) (recogniser recognitionport.Recogniser, closer func() error, why error) {
	host, err := c.resolvePlugin(bridge.CapabilityRecognition)
	if err != nil {
		return nil, nil, err
	}
	bridge, err := recogniserbridge.Open(ctx, host)
	if err != nil {
		return nil, nil, err
	}
	return bridge, bridge.Close, nil
}

// Recognise is one document read with the recogniser given.
func (c Config) Recognise(sources sourceport.SourceRepository, recogniser recognitionport.Recogniser) recognition.Recognise {
	return recognition.NewRecognise(c.VaultReaders(), sources, c.GetDerivedStores(), c.PageRenderer(), recogniser)
}

// OpenRecognitionWorker is the queue that reads this installation's scanned documents.
func (c Config) OpenRecognitionWorker(
	ctx context.Context,
	sources sourceport.SourceRepository,
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
			Open: func(ctx context.Context, _ func(what string, finished, total int64)) (recognitionport.Recogniser, func() error, error) {
				return c.Recogniser(ctx)
			},
			Ready:    c.RecogniserReady,
			Prepared: c.RecogniserReady,
		},
		Proofreading: c.getProofreading(c.ScanProofreading),
	})
}

// PageRenderer draws the pages of a document through the document plugin, looked up when it is
// asked.
func (c Config) PageRenderer() sourceport.PageRenderer {
	host, err := c.resolvePlugin(bridge.CapabilityDocument)
	if err != nil {
		return unavailableRenderer{why: err}
	}
	return pluginRenderer{host: host}
}

// pluginRenderer reaches the document plugin. A document it opens holds its own reference on the
// plugin.
type pluginRenderer struct {
	host pluginport.PluginManager
}

func (r pluginRenderer) OpenDocument(ctx context.Context, file io.ReadSeekCloser) (sourceport.OpenDocument, error) {
	bridge, err := documentbridge.Open(ctx, r.host)
	if err != nil {
		if file != nil {
			_ = file.Close()
		}
		return nil, err
	}
	defer bridge.Close()
	return bridge.OpenDocument(ctx, file)
}

// unavailableRenderer answers every ask with why no document plugin is there.
type unavailableRenderer struct {
	why error
}

func (u unavailableRenderer) OpenDocument(_ context.Context, file io.ReadSeekCloser) (sourceport.OpenDocument, error) {
	if file != nil {
		_ = file.Close()
	}
	return nil, u.why
}
