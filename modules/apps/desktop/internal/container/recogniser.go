package container

import (
	"context"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/ocr/onnx"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/task"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/source"
)

// Recogniser is what reads a scanned page on this machine, opened now.
//
// Three values, as with the embedder: the recogniser, what gives it back, and
// why there is none. A failure is absence and not an error — recognition is
// unavailable and everything else works.
//
// It waits for whatever is missing, so it is for a terminal, where waiting is
// what a person came for. A window asks Recognising instead.
func (c Config) Recogniser() (recogniser port.Recogniser, close func() error, why error) {
	models, err := onnx.Open(context.Background(), c.Recognition)
	if err != nil {
		return nil, nil, err
	}
	return models, models.Close, nil
}

// reading is what the one piece of work this runs is called, wherever it is
// shown. The same work reported again replaces itself.
const reading = "reading"

// Recognising reads scanned documents behind whoever asked.
//
// Nothing here is done inside the question that asked for it. Fetching the
// models is minutes and reading a book is an hour, and an answer that arrives in
// an hour is a program that has hung. The ask starts the work and says so, and
// how far it has got is put where everything else being done is put.
type Recognising struct {
	cfg     Config
	sources port.SourceRepository
	tasks   *task.Tasks

	mu      sync.Mutex
	running bool
}

// Recognising is the recogniser this installation offers, reporting itself into
// the list of what is being done.
func (c Config) Recognising(sources port.SourceRepository, tasks *task.Tasks) *Recognising {
	return &Recognising{cfg: c, sources: sources, tasks: tasks}
}

// Ready says whether a document could be read now without waiting for anything
// to arrive.
func (r *Recognising) Ready() bool { return onnx.Ready(r.cfg.Recognition) }

// Running says whether a document is being read.
func (r *Recognising) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Start begins reading one document behind whoever asked, and says whether it
// began.
//
// One at a time: the models hold a worker each, and a second reading would take
// twice as long and say so half as clearly.
//
// The context is the application's rather than the caller's, because whoever
// asked is answered at once and goes away.
func (r *Recognising) Start(ctx context.Context, v domain.Vault, path string) bool {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return false
	}
	r.running = true
	r.mu.Unlock()

	r.say(task.Task{ID: reading, Doing: "Reading a scan", About: path})

	go func() {
		err := r.read(ctx, v, path)

		r.mu.Lock()
		r.running = false
		r.mu.Unlock()

		if err != nil {
			// A failure nobody was shown is a failure nobody can act on, so it
			// stays in the list until the next reading replaces it.
			r.say(task.Task{ID: reading, Doing: "Reading a scan", About: path, Failed: err.Error()})
			return
		}
		r.done()
	}()
	return true
}

// read is the work itself: what is missing arrives, and then the document is
// read.
func (r *Recognising) read(ctx context.Context, v domain.Vault, path string) error {
	fetching := r.cfg.Recognition
	fetching.Fetching = func(what string, done, total int64) {
		// Counted in megabytes because that is the size a person reads. Bytes
		// are nine digits and say nothing that the first three do not.
		r.say(task.Task{
			ID: reading, Doing: "Fetching models", About: what,
			Done: done >> 20, Total: total >> 20,
		})
	}

	models, err := onnx.Open(ctx, fetching)
	if err != nil {
		return fmt.Errorf("nothing to read with: %w", err)
	}
	defer models.Close()

	r.say(task.Task{ID: reading, Doing: "Reading a scan", About: path})
	_, err = source.Recognise{
		Readers: r.cfg.VaultReaders(),
		Sources: r.sources,
		Derived: r.cfg.DerivedStores(),
		By:      models,
		OnProgress: func(res source.RecogniseResult) {
			r.say(task.Task{
				ID:    reading,
				Doing: "Reading a scan",
				About: path,
				Done:  int64(res.Read),
				Total: int64(res.Pages),
			})
		},
	}.Execute(ctx, v, path)
	return err
}

func (r *Recognising) say(at task.Task) {
	if r.tasks != nil {
		r.tasks.Set(at)
	}
}

func (r *Recognising) done() {
	if r.tasks != nil {
		r.tasks.Done(reading)
	}
}
