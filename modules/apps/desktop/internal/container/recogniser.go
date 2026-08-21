package container

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

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

// reading is what one reading is called, wherever it is shown. It stands for
// the whole of that reading, so what it reports again replaces itself, and one
// reading dismissed is one reading dismissed.
func reading() string { return fmt.Sprintf("reading-%d", time.Now().UnixNano()) }

// correcting is what one reading's proofreading is called, wherever it is
// shown. One reading is one line, and it replaces itself as pages are put right.
func correcting(path string) string { return "proofreading-" + path }

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

	// Cut makes a source's chunks from what has been read of it. It is called
	// as pages are written down, so a page is searchable when it is read.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	mu      sync.Mutex
	running bool
	// last is what the reading before this one was called. A reading that
	// failed is left in the list under that name, and the next reading takes it
	// out.
	last string
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
	before := r.last
	id := reading()
	r.last = id
	r.mu.Unlock()

	r.done(before)
	r.say(task.Task{ID: id, Doing: "Reading a scan", About: path})

	go func() {
		err := r.read(ctx, v, id, path)
		if err == nil {
			r.correct(ctx, v, path)
		}

		switch {
		case err == nil, errors.Is(err, context.Canceled):
			// A reading somebody stopped is a reading that is over.
			r.done(id)
		default:
			// A failure nobody was shown is a failure nobody can act on, so it
			// stays in the list until it is dismissed or the next reading
			// begins.
			r.say(task.Task{ID: id, Doing: "Reading a scan", About: path, Failed: err.Error()})
		}

		// The task is finished before the run is, so that a reading begun the
		// moment this one ends has the list to itself.
		r.mu.Lock()
		r.running = false
		r.mu.Unlock()
	}()
	return true
}

// read is the work itself: what is missing arrives, and then the document is
// read.
func (r *Recognising) read(ctx context.Context, v domain.Vault, id, path string) error {
	fetching := r.cfg.Recognition
	fetching.Fetching = func(what string, done, total int64) {
		// Counted in megabytes because that is the size a person reads. Bytes
		// are nine digits and say nothing that the first three do not.
		r.say(task.Task{
			ID: id, Doing: "Fetching models", About: what,
			Done: done >> 20, Total: total >> 20,
		})
	}

	models, err := onnx.Open(ctx, fetching)
	if err != nil {
		return fmt.Errorf("nothing to read with: %w", err)
	}
	defer models.Close()

	r.say(task.Task{ID: id, Doing: "Reading a scan", About: path})
	res, err := source.Recognise{
		Readers: r.cfg.VaultReaders(),
		Sources: r.sources,
		Derived: r.cfg.DerivedStores(),
		By:      models,
		Cut:     r.Cut,
		OnProgress: func(res source.RecogniseResult) {
			r.say(task.Task{
				ID:    id,
				Doing: "Reading a scan",
				About: path,
				Done:  int64(res.Read),
				Total: int64(res.Pages),
			})
		},
	}.Execute(ctx, v, path)
	if err != nil {
		return err
	}
	if res.Busy {
		// Another run holds these bytes — a terminal, or a second window. What
		// it reads is what this would have read, and saying so is what the
		// person is owed.
		return fmt.Errorf("%s is already being read", path)
	}
	return nil
}

// correct puts a reading right, where a person configured something to
// proofread it with. An installation that named none does nothing here.
//
// It reports itself under its own name, and a reading whose proofreading failed
// is the reading as it was read.
func (r *Recognising) correct(ctx context.Context, v domain.Vault, path string) {
	by, err := r.cfg.Proofreader()
	if err != nil {
		r.say(task.Task{ID: correcting(path), Doing: "Proofreading a reading", About: path, Failed: err.Error()})
		return
	}
	if by == nil {
		return
	}

	// A proofreader with a queue is left the pages and answers later, and the
	// batch is collected by whatever comes back for it.
	queue, err := r.cfg.ProofreadQueue()
	if err != nil {
		return
	}

	id := correcting(path)
	service := r.cfg.Proofreading.Service
	r.say(task.Task{ID: id, Doing: "Proofreading a reading", About: path})

	_, err = source.Proofread{
		Readers: r.cfg.VaultReaders(),
		Derived: r.cfg.DerivedStores(),
		By:      by,
		Queue:   queue,
		Pages:   service.PagesAtOnce,
		Apart:   service.LettersApart,
		Cut:     r.Cut,
		OnProgress: func(res source.ProofreadResult) {
			r.say(task.Task{
				ID:    id,
				Doing: "Proofreading a reading",
				About: path,
				Done:  int64(res.Read),
				Total: int64(res.Pages),
			})
		},
	}.Execute(ctx, v, path)

	switch {
	case err == nil, errors.Is(err, context.Canceled):
		r.done(id)
	default:
		r.say(task.Task{ID: id, Doing: "Proofreading a reading", About: path, Failed: err.Error()})
	}
}

// say puts this reading in the list of what is being done. A person asked for
// it and is waiting to be told it began.
func (r *Recognising) say(at task.Task) {
	at.Asked = true
	if r.tasks != nil {
		r.tasks.Set(at)
	}
}

func (r *Recognising) done(id string) {
	if r.tasks != nil {
		r.tasks.Done(id)
	}
}

// Collecting asks after the batches left with a proofreader, until the context
// is done. A batch outlives the run that left it, so one left before the
// application closed is collected when it opens.
//
// Nothing here is done unless a person configured a proofreader with a queue.
func (r *Recognising) Collecting(
	ctx context.Context,
	known port.SourceQueries,
	every time.Duration,
	vaults ...domain.Vault,
) {
	queue, err := r.cfg.ProofreadQueue()
	if err != nil || queue == nil {
		return
	}
	for {
		for _, v := range vaults {
			r.collect(ctx, known, queue, v)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(every):
		}
	}
}

// collect takes up every reading of one vault that has a batch out.
func (r *Recognising) collect(
	ctx context.Context,
	known port.SourceQueries,
	queue port.ProofreadQueue,
	v domain.Vault,
) {
	read, err := known.Recognised(ctx, v.ID, domain.KindBook)
	if err != nil {
		return
	}
	service := r.cfg.Proofreading.Service
	for _, said := range read {
		if ctx.Err() != nil {
			return
		}
		id := correcting(said.Path)
		res, err := source.Proofread{
			Readers: r.cfg.VaultReaders(),
			Derived: r.cfg.DerivedStores(),
			By:      queue,
			Queue:   queue,
			Pages:   service.PagesAtOnce,
			Apart:   service.LettersApart,
			Cut:     r.Cut,
		}.Execute(ctx, v, said.Path)

		switch {
		case err != nil:
			r.say(task.Task{
				ID: id, Doing: "Proofreading a reading",
				About: said.Path, Failed: err.Error(),
			})
		case res.None, res.Busy, res.Read >= res.Pages:
			// A reading with nothing left to put right is a reading nobody is
			// waiting on.
			r.done(id)
		default:
			r.say(task.Task{
				ID: id, Doing: "Proofreading a reading", About: said.Path,
				Done: int64(res.Read), Total: int64(res.Pages),
			})
		}
	}
}
