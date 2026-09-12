package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// RecogniserReady says whether a document could be read now without waiting for
// anything to arrive. Which models those are is settled here, with every other
// choice of adapter.
func (c Config) RecogniserReady() bool { return recognition.Ready(c.Recognition) }

// PrepareRecogniser makes the runtime this process reads a page through, and is
// called before a window is made. One made after a window reads every page it is
// given as nothing.
//
// A machine holding no runtime says so and is left as it is: reading is what
// fetches one.
func (c Config) PrepareRecogniser(ctx context.Context) error {
	return recognition.Prepare(ctx, c.Recognition)
}

// Recogniser is what reads a scanned page on this machine, opened now: the
// recogniser, what gives it back, and why there is none.
//
// It waits for whatever is missing, so it is for a terminal, where waiting is
// what a person came for. A window asks Recognising instead.
func (c Config) Recogniser(ctx context.Context) (recogniser port.Recogniser, close func() error, why error) {
	models, err := recognition.Open(ctx, c.Recognition)
	if err != nil {
		return nil, nil, err
	}
	return models, models.Close, nil
}

// Recognise is one document read with the recogniser given, for a caller that
// asked for that document and waits for it. Recognising is the queue that reads
// what nobody asked about.
func (c Config) Recognise(sources port.SourceRepository, by port.Recogniser) source.Recognise {
	return source.NewRecognise(c.VaultReaders(), sources, c.GetDerivedStores(), c.PageRenderer(), by)
}

// OpenRecognitionWorker is the queue that reads this installation's scanned
// documents, built against the adapters it was configured with and reporting
// itself into the list of what is being done.
//
// models is what it takes its turn at with the queue that listens to
// recordings: one run holds this machine's models at a time.
func (c Config) OpenRecognitionWorker(
	ctx context.Context,
	sources port.SourceRepository,
	tasks *task.Tasks,
	models *source.Lock,
) *source.RecognitionWorker {
	return source.NewRecognitionWorker(ctx, source.Recognitions{
		Readers:   c.VaultReaders(),
		Derived:   c.GetDerivedStores(),
		Documents: c.PageRenderer(),
		Sources:   sources,
		Tasks:     tasks,
		Models:    models,
		Runtime: source.RecognitionRuntime{
			Open: func(
				ctx context.Context, tell func(what string, done, total int64),
			) (port.Recogniser, func() error, error) {
				cfg := c.Recognition
				cfg.Progress = tell
				models, err := recognition.Open(ctx, cfg)
				if err != nil {
					return nil, nil, err
				}
				return models, models.Close, nil
			},
			Ready:    c.RecogniserReady,
			Prepared: recognition.IsPrepared,
		},
		Proofreading: c.getProofreading(c.ScanProofreading),
	})
}
