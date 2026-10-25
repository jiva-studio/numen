package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/recogniserbridge"
	"github.com/jiva-studio/numen/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/libs/core/usecase/run"

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
