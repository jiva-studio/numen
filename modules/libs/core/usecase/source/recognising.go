package source

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// errLateRuntime is a document left unread because what would read it was
// fetched after the window was made.
var errLateRuntime = errors.New("what reads a scan arrived just now; open numen again to read it")

// readingLine is what one reading is called, wherever it is shown. It stands for
// the whole of that reading, so what it reports again replaces itself, and one
// reading dismissed is one reading dismissed.
func readingLine() string { return fmt.Sprintf("reading-%d", time.Now().UnixNano()) }

// correctingLine is what putting one file's text right is called, wherever it is
// shown. One file is one line, and it replaces itself as the text is put right.
func correctingLine(path string) string { return "proofreading-" + path }

// Reads is what reads a scanned page, opened when there is one to read. It is
// told how far the fetching of what it needs has got.
type Reads func(ctx context.Context, tell func(what string, done, total int64)) (port.Recogniser, func() error, error)

// Readings is what one installation reads scanned documents with: what reads a
// page, what a vault is read and written through, and what puts a reading right
// afterwards.
type Readings struct {
	Readers   port.VaultReaders
	Derived   port.DerivedStores
	Documents port.Documents
	Sources   port.SourceRepository
	Tasks     *task.Tasks

	// Open reads a page, and Ready says whether opening it would wait for
	// anything to arrive.
	Open  Reads
	Ready func() bool

	// Standing says whether the runtime a page is read through was made before
	// the window. A test that reads through nothing leaves it unset.
	Standing func() bool

	// Proofreading is what a reading is put right with.
	Proofreading Correcting
}

// Recognising reads scanned documents behind whoever asked.
//
// Nothing here is done inside the question that asked for it. Fetching the
// models is minutes and reading a book is an hour, and an answer that arrives in
// an hour is a program that has hung. The ask starts the work and says so, and
// how far it has got is put where everything else being done is put.
type Recognising struct {
	with Readings

	// under is what every reading runs under, and going is every reading that
	// has not ended. A reading outlives the question that asked for it and ends
	// with the application, which waits here for what it is still writing.
	under context.Context
	going sync.WaitGroup

	// Cut makes a source's chunks from what has been read of it. It is called
	// as pages are written down, so a page is searchable when it is read.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	mu      sync.Mutex
	running bool
	// asked is the documents a person named that have not been read yet.
	asked asked
	// last is what the reading before this one was called. A reading that
	// failed is left in the list under that name, and the next reading takes it
	// out.
	last string
}

// NewRecognising is the recogniser an installation offers, reporting itself into
// the list of what is being done.
func NewRecognising(ctx context.Context, with Readings) *Recognising {
	return &Recognising{with: with, under: ctx}
}

// Ready says whether a document could be read now without waiting for anything
// to arrive.
func (r *Recognising) Ready() bool { return r.with.Ready() }

// Wait is every reading and every collection this started, ended. What they
// write goes into an index the application still holds open.
func (r *Recognising) Wait() { r.going.Wait() }

// Running says whether a document is being read.
func (r *Recognising) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Start reads one document a person named, and says whether it began now or
// waits its turn.
//
// One at a time: the models hold a worker each, and a second reading would take
// twice as long and say so half as clearly. A document named while one is being
// read goes to the back of the line and is read as soon as the turn is free.
//
// It runs under the application, so whoever asked is answered at once and goes
// away while the reading carries on.
func (r *Recognising) Start(v domain.Vault, path string) port.Taking {
	r.mu.Lock()
	r.asked.want(v, path)
	if r.running {
		r.mu.Unlock()
		return port.Queued
	}
	r.running = true
	r.mu.Unlock()

	r.going.Add(1)
	go func() {
		defer r.going.Done()
		defer r.stopped()
		r.drain(r.context())
	}()
	return port.Began
}

// Waiting is how many documents a person named are still in line.
func (r *Recognising) Waiting() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.asked.waiting()
}

// drain reads every document a person named, in the order they named them. A
// document is out of the line before it is read, so one whose reading ends
// where nothing expected it to holds nothing afterwards.
func (r *Recognising) drain(ctx context.Context) {
	for {
		r.mu.Lock()
		one, waiting := r.asked.take()
		r.mu.Unlock()
		if !waiting || ctx.Err() != nil {
			return
		}
		r.one(ctx, one.vault, one.path)
	}
}

// one is a single document read, put right, and reported.
func (r *Recognising) one(ctx context.Context, v domain.Vault, path string) {
	r.mu.Lock()
	before := r.last
	id := readingLine()
	r.last = id
	r.mu.Unlock()

	r.done(before)
	r.say(task.Task{ID: id, Doing: "Reading a scan", About: path})

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
		// stays in the list until it is dismissed or the next reading begins.
		r.say(task.Task{ID: id, Doing: "Reading a scan", About: path, Failed: err.Error()})
	}
}

func (r *Recognising) stopped() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running = false
}

func (r *Recognising) context() context.Context {
	if r.under == nil {
		return context.Background()
	}
	return r.under
}

// read is the work itself: what is missing arrives, and then the document is
// read.
func (r *Recognising) read(ctx context.Context, v domain.Vault, id, path string) error {
	// One heavy run on a machine: a recording being heard holds the turn, and
	// this waits for it.
	// A scan is read only where somebody asked for it.
	turn, err := heavy.take(ctx, true, func() {
		r.say(task.Task{ID: id, Doing: "Waiting for a turn at the models", About: path})
	})
	if err != nil {
		return err
	}
	defer turn()

	// Getting the models is a step of its own and stands under its own name.
	// Which file is coming down, and how much of it, is known once one is.
	r.say(task.Task{ID: id, Doing: "Fetching models"})
	models, close, err := r.with.Open(ctx, func(what string, done, total int64) {
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
	if r.with.Standing != nil && !r.with.Standing() {
		return errLateRuntime
	}

	r.say(task.Task{ID: id, Doing: "Reading a scan", About: path})
	res, err := Recognise{
		Readers:   r.with.Readers,
		Sources:   r.with.Sources,
		Derived:   r.with.Derived,
		Documents: r.with.Documents,
		By:        models,
		Cut:       r.Cut,
		OnProgress: func(res RecogniseResult) {
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
// proofread it with. An installation that named no profile, or asked for a
// reading to be put right by hand, does nothing here.
//
// It reports itself under its own name, and a reading whose proofreading failed
// is the reading as it was read.
func (r *Recognising) correct(ctx context.Context, v domain.Vault, path string) {
	said := r.with.Proofreading
	if !said.Automatically {
		return
	}

	by, err := said.By(proofread.ScanInstruction)
	if err != nil {
		r.say(task.Task{ID: correctingLine(path), Doing: "Proofreading a reading", About: path, Failed: err.Error()})
		return
	}
	if by == nil {
		return
	}

	// A proofreader with a queue is left the pages and answers later, and the
	// batch is collected by whatever comes back for it.
	queue, err := said.Queue(proofread.ScanInstruction)
	if err != nil {
		r.say(task.Task{ID: correctingLine(path), Doing: "Proofreading a reading", About: path, Failed: err.Error()})
		return
	}

	id := correctingLine(path)
	r.say(task.Task{ID: id, Doing: "Proofreading a reading", About: path})

	_, err = Proofread{
		Readers:         r.with.Readers,
		Derived:         r.with.Derived,
		By:              by,
		Queue:           queue,
		Pages:           said.Batch,
		MaxEditDistance: said.MaxEditDistance,
		Cut:             r.Cut,
		OnProgress: func(res ProofreadResult) {
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
func (r *Recognising) say(at task.Task) { r.says(at, true) }

// says puts one piece of work in the list. Work a person started is shown at
// once, and work nobody asked for is shown once it has lasted.
func (r *Recognising) says(at task.Task, asked bool) {
	at.Asked = asked
	if r.with.Tasks != nil {
		r.with.Tasks.Set(at)
	}
}

func (r *Recognising) done(id string) {
	if r.with.Tasks != nil {
		r.with.Tasks.Done(id)
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
	queue, err := r.with.Proofreading.Queue(proofread.ScanInstruction)
	if err != nil || queue == nil {
		return
	}
	r.going.Add(1)
	defer r.going.Done()
	for {
		for _, v := range vaults {
			r.collect(ctx, known, queue, queue, v)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(every):
		}
	}
}

// TakingUp puts right the readings of these vaults that stand short of their
// last page, once, behind the caller.
//
// A proofreading stands at the page it reached, so a run that ended among the
// batches is taken up at that page. A reading no proofreader has been over
// stands at its first page and is put right whole. A proofreader with a queue
// leaves a batch behind it and is taken up by Collecting.
func (r *Recognising) TakingUp(
	ctx context.Context,
	known port.SourceQueries,
	vaults ...domain.Vault,
) {
	said := r.with.Proofreading
	if !said.Automatically {
		return
	}
	r.going.Add(1)
	go func() {
		defer r.going.Done()
		queue, err := said.Queue(proofread.ScanInstruction)
		if err != nil || queue != nil {
			return
		}
		by, err := said.By(proofread.ScanInstruction)
		if err != nil || by == nil {
			return
		}
		for _, v := range vaults {
			r.collect(ctx, known, by, nil, v)
		}
	}()
}

// collect takes up every reading of one vault that stands short of its last
// page.
func (r *Recognising) collect(
	ctx context.Context,
	known port.SourceQueries,
	by port.Proofreader,
	queue port.ProofreadQueue,
	v domain.Vault,
) {
	read, err := known.Recognised(ctx, v.ID, domain.KindBook)
	if err != nil {
		return
	}
	said := r.with.Proofreading
	for _, one := range read {
		if ctx.Err() != nil {
			return
		}
		id := correctingLine(one.Path)
		res, err := Proofread{
			Readers:         r.with.Readers,
			Derived:         r.with.Derived,
			By:              by,
			Queue:           queue,
			Pages:           said.Batch,
			MaxEditDistance: said.MaxEditDistance,
			Cut:             r.Cut,
			OnProgress: func(res ProofreadResult) {
				r.says(task.Task{
					ID: id, Doing: "Proofreading a reading", About: one.Path,
					Done: int64(res.Read), Total: int64(res.Pages),
				}, false)
			},
		}.Execute(ctx, v, one.Path)

		switch {
		case err != nil:
			r.says(task.Task{
				ID: id, Doing: "Proofreading a reading",
				About: one.Path, Failed: err.Error(),
			}, false)
		case res.Busy:
			// The reading is held by another run, and that run is the one whose
			// progress the list carries.
		case res.None, res.Read >= res.Pages:
			// A reading with nothing left to put right is a reading nobody is
			// waiting on.
			r.done(id)
		default:
			r.says(task.Task{
				ID: id, Doing: "Proofreading a reading", About: one.Path,
				Done: int64(res.Read), Total: int64(res.Pages),
			}, false)
		}
	}
}
