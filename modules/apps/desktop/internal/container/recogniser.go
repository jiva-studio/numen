package container

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/recognition"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/task"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/source"
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

// errLateRuntime is a document left unread because what would read it was
// fetched after the window was made.
var errLateRuntime = errors.New("what reads a scan arrived just now; open numen again to read it")

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

	// under is what every reading runs under, and going is every reading that
	// has not ended. A reading outlives the question that asked for it and ends
	// with the application, which waits here for what it is still writing.
	under context.Context
	going sync.WaitGroup

	// open is what reads a page, opened when there is one to read, and ready
	// says whether opening it would wait for anything to arrive. Which models
	// those are is settled where every other adapter is chosen.
	open  opening
	ready func() bool

	// standing says whether the runtime a page is read through was made before
	// the window. A test that reads through nothing leaves it unset.
	standing func() bool

	// queue is where pages are left for a proofreader to answer about later,
	// built when there are pages to leave.
	queue func() (port.ProofreadQueue, error)

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
func (c Config) Recognising(ctx context.Context, sources port.SourceRepository, tasks *task.Tasks) *Recognising {
	return &Recognising{
		cfg: c, sources: sources, tasks: tasks, under: ctx,
		open: func(ctx context.Context, tell func(what string, done, total int64)) (port.Recogniser, func() error, error) {
			cfg := c.Recognition
			cfg.Fetching = tell
			models, err := recognition.Open(ctx, cfg)
			if err != nil {
				return nil, nil, err
			}
			return models, models.Close, nil
		},
		ready:    c.RecogniserReady,
		standing: recognition.Prepared,
		queue:    c.ProofreadQueue,
	}
}

// opening is what reads a scanned page, opened when there is one to read. It is
// told how far the fetching of what it needs has got.
type opening func(ctx context.Context, tell func(what string, done, total int64)) (port.Recogniser, func() error, error)

// Ready says whether a document could be read now without waiting for anything
// to arrive.
func (r *Recognising) Ready() bool { return r.ready() }

// Wait is every reading and every collection this started, ended. What they
// write goes into an index the application still holds open.
func (r *Recognising) Wait() { r.going.Wait() }

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
// It runs under the application, so whoever asked is answered at once and goes
// away while the reading carries on.
func (r *Recognising) Start(v domain.Vault, path string) bool {
	ctx := r.under
	if ctx == nil {
		ctx = context.Background()
	}
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

	r.going.Add(1)
	go func() {
		defer r.going.Done()
		// The task is finished before the run is, so that a reading begun the
		// moment this one ends has the list to itself. It is finished however
		// this reading ends.
		defer func() {
			r.mu.Lock()
			r.running = false
			r.mu.Unlock()
		}()

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
	}()
	return true
}

// read is the work itself: what is missing arrives, and then the document is
// read.
func (r *Recognising) read(ctx context.Context, v domain.Vault, id, path string) error {
	models, close, err := r.open(ctx, func(what string, done, total int64) {
		// The count is bytes and says so, and the sizes a person reads them in
		// are the window's to write.
		r.say(task.Task{
			ID: id, Doing: "Fetching models", About: what,
			Done: done, Total: total, Counting: task.Bytes,
		})
	})
	if err != nil {
		return fmt.Errorf("nothing to read with: %w", err)
	}
	defer close()

	// Every page of this process is read through the runtime it made before its
	// window, and a runtime this process fetched afterwards is not that one. What
	// was missing is here now, and the reading is the next opening's to do.
	if r.standing != nil && !r.standing() {
		return errLateRuntime
	}

	r.say(task.Task{ID: id, Doing: "Reading a scan", About: path})
	res, err := source.Recognise{
		Readers:   r.cfg.VaultReaders(),
		Sources:   r.sources,
		Derived:   r.cfg.DerivedStores(),
		Documents: r.cfg.Documents(),
		By:        models,
		Cut:       r.Cut,
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
		// it reads is what this reads.
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
	queue, err := r.queue()
	if err != nil {
		r.say(task.Task{ID: correcting(path), Doing: "Proofreading a reading", About: path, Failed: err.Error()})
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
	queue, err := r.queue()
	if err != nil || queue == nil {
		return
	}
	r.going.Add(1)
	defer r.going.Done()
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
