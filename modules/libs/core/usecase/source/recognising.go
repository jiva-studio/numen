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

// errLateRuntime is a document left unrecognised because what would recognise
// it was fetched after the window was made.
var errLateRuntime = errors.New("what reads a scan arrived just now; open numen again to read it")

// recognitionID is what one recognition is called, wherever it is shown. It
// stands for the whole of that recognition, so what it reports again replaces
// itself, and one recognition dismissed is one recognition dismissed.
func recognitionID() string { return fmt.Sprintf("recognition-%d", time.Now().UnixNano()) }

// proofreadingID is what putting one file's text right is called, wherever it is
// shown. One file is one line, and it replaces itself as the text is put right.
func proofreadingID(path string) string { return "proofreading-" + path }

// Recognises is what recognises a scanned page, opened when there is one to
// recognise. It is told how far the fetching of what it needs has got.
type Recognises func(ctx context.Context, tell func(what string, done, total int64)) (port.Recogniser, func() error, error)

// Recognitions is what one installation recognises scanned documents with: what
// recognises a page, what a vault is read and written through, and what puts a
// recognition right afterwards.
type Recognitions struct {
	Readers   port.VaultReaders
	Derived   port.DerivedStores
	Documents port.Documents
	Sources   port.SourceRepository
	Tasks     *task.Tasks

	// Open recognises a page, and Ready says whether opening it would wait for
	// anything to arrive.
	Open  Recognises
	Ready func() bool

	// Standing says whether the runtime a page is recognised through was made
	// before the window. A test that recognises through nothing leaves it unset.
	Standing func() bool

	// Proofreading is what a recognition is put right with.
	Proofreading Proofreading
}

// Recognising recognises scanned documents behind whoever asked.
//
// Nothing here is done inside the question that asked for it. Fetching the
// models is minutes and recognising a book is an hour, and an answer that
// arrives in an hour is a program that has hung. The ask starts the work and
// says so, and how far it has got is put where everything else being done is
// put.
type Recognising struct {
	with Recognitions

	// under is what every recognition runs under, and going is every recognition
	// that has not ended. A recognition outlives the question that asked for it
	// and ends with the application, which waits here for what it is still
	// writing.
	under context.Context
	going sync.WaitGroup

	// Cut makes a source's chunks from what has been recognised of it. It is
	// called as pages are written down, so a page is searchable when it is
	// recognised.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	mu      sync.Mutex
	running bool
	// runs counts the runs that have begun. A run stops the running only while
	// it is still the one running.
	runs uint64
	// queue is the documents a person named that have not been recognised yet.
	queue queue
	// idle is called where a run has found the line empty and stopped. A test
	// names a document there, at the one instant the two could cross.
	idle func()
	// last is what the recognition before this one was called. A recognition
	// that failed is left in the list under that name, and the next recognition
	// takes it out.
	last string
}

// NewRecognising is the recogniser an installation offers, reporting itself into
// the list of what is being done.
func NewRecognising(ctx context.Context, with Recognitions) *Recognising {
	return &Recognising{with: with, under: ctx}
}

// Ready says whether a document could be recognised now without waiting for
// anything to arrive.
func (r *Recognising) Ready() bool { return r.with.Ready() }

// Wait is every recognition and every collection this started, ended. What they
// write goes into an index the application still holds open.
func (r *Recognising) Wait() { r.going.Wait() }

// Running says whether a document is being recognised.
func (r *Recognising) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Start recognises one document a person named, and says whether it began now
// or waits its turn.
//
// One at a time: the models hold a worker each, and a second recognition would
// take twice as long and say so half as clearly. A document named while one is
// being recognised goes to the back of the line and is recognised as soon as
// the run before it ends.
//
// It runs under the application, so whoever asked is answered at once and goes
// away while the recognition carries on.
func (r *Recognising) Start(v domain.Vault, path string) port.Taking {
	r.mu.Lock()
	r.queue.add(v, path)
	if r.running {
		r.mu.Unlock()
		return port.Queued
	}
	r.running = true
	r.runs++
	mine := r.runs
	r.mu.Unlock()

	r.going.Add(1)
	go func() {
		defer r.going.Done()
		defer r.stopped(mine)
		r.drain(r.context())
	}()
	return port.Began
}

// Waiting is how many documents a person named are still in line.
func (r *Recognising) Waiting() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.queue.waiting()
}

// drain recognises every document a person named, in the order they named them.
// A document is out of the line before it is recognised, so one whose
// recognition ends where nothing expected it to holds nothing afterwards.
// The line is found empty and the running stopped under one hold of the lock,
// so a document named at that instant is answered "began" and recognised by the
// run that answers it.
func (r *Recognising) drain(ctx context.Context) {
	for {
		r.mu.Lock()
		one, waiting := r.queue.take()
		if !waiting || ctx.Err() != nil {
			r.running = false
			r.mu.Unlock()
			if r.idle != nil {
				r.idle()
			}
			return
		}
		r.mu.Unlock()
		r.one(ctx, one.vault, one.path)
	}
}

// one is a single document recognised, put right, and reported.
func (r *Recognising) one(ctx context.Context, v domain.Vault, path string) {
	r.mu.Lock()
	before := r.last
	id := recognitionID()
	r.last = id
	r.mu.Unlock()

	r.done(before)
	r.say(task.Task{ID: id, Doing: "Reading a scan", About: path})

	err := r.recognise(ctx, v, id, path)
	if err == nil {
		r.proofread(ctx, v, path)
	}

	switch {
	case err == nil, errors.Is(err, context.Canceled):
		// A recognition somebody stopped is a recognition that is over.
		r.done(id)
	default:
		// A failure nobody was shown is a failure nobody can act on, so it
		// stays in the list until it is dismissed or the next recognition
		// begins.
		r.say(task.Task{ID: id, Doing: "Reading a scan", About: path, Failed: err.Error()})
	}
}

// stopped stops the running where this run is still the one running, so a
// recognition that ended where nothing expected it to leaves nothing running.
func (r *Recognising) stopped(run uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runs == run {
		r.running = false
	}
}

func (r *Recognising) context() context.Context {
	if r.under == nil {
		return context.Background()
	}
	return r.under
}

// recognise is the work itself: what is missing arrives, and then the document
// is recognised.
func (r *Recognising) recognise(ctx context.Context, v domain.Vault, id, path string) error {
	// One run holds the models on a machine: a recording being transcribed holds
	// them, and this waits for it.
	// A scan is recognised only where somebody asked for it.
	release, err := models.acquire(ctx, true, func() {
		r.say(task.Task{ID: id, Doing: "Waiting for the models", About: path})
	})
	if err != nil {
		return err
	}
	defer release()

	// Getting the models is a step of its own and stands under its own name.
	// Which file is coming down, and how much of it, is known once one is.
	r.say(task.Task{ID: id, Doing: "Fetching models"})
	by, close, err := r.with.Open(ctx, func(what string, done, total int64) {
		// The count is bytes and says so, and the sizes a person reads them in
		// are the window's to write.
		r.say(task.Task{
			ID: id, Doing: "Fetching models", About: what,
			Done: done, Total: total, Counting: task.Bytes,
		})
	})
	if err != nil {
		return fmt.Errorf("nothing to recognise with: %w", err)
	}
	defer close()

	// Every page of this process is recognised through the runtime it made before
	// its window, and a runtime this process fetched afterwards is not that one.
	// What was missing is here now, and the recognition is the next opening's to
	// do.
	if r.with.Standing != nil && !r.with.Standing() {
		return errLateRuntime
	}

	r.say(task.Task{ID: id, Doing: "Reading a scan", About: path})
	res, err := Recognise{
		Readers:   r.with.Readers,
		Sources:   r.with.Sources,
		Derived:   r.with.Derived,
		Documents: r.with.Documents,
		By:        by,
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
		// it recognises is what this recognises.
		return fmt.Errorf("%s is already being read", path)
	}
	return nil
}

// proofread puts a recognition right, where a person configured something to
// proofread it with. An installation that named no profile, or asked for a
// recognition to be put right by hand, does nothing here.
//
// It reports itself under its own name, and a recognition whose proofreading
// failed is the recognition as it was recognised.
func (r *Recognising) proofread(ctx context.Context, v domain.Vault, path string) {
	said := r.with.Proofreading
	if !said.Automatically {
		return
	}

	by, err := said.By(proofread.ScanInstruction)
	if err != nil {
		r.say(task.Task{ID: proofreadingID(path), Doing: "Proofreading a reading", About: path, Failed: err.Error()})
		return
	}
	if by == nil {
		return
	}

	// A proofreader with a queue is left the pages and answers later, and the
	// batch is collected by whatever comes back for it.
	queue, err := said.Queue(proofread.ScanInstruction)
	if err != nil {
		r.say(task.Task{ID: proofreadingID(path), Doing: "Proofreading a reading", About: path, Failed: err.Error()})
		return
	}

	id := proofreadingID(path)
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

// say puts this recognition in the list of what is being done. A person asked
// for it and is waiting to be told it began.
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

// TakingUp puts right the recognitions of these vaults that stand short of
// their last page, once, behind the caller.
//
// A proofreading stands at the page it reached, so a run that ended among the
// batches is taken up at that page. A recognition no proofreader has been over
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

// collect takes up every recognition of one vault that stands short of its last
// page.
func (r *Recognising) collect(
	ctx context.Context,
	known port.SourceQueries,
	by port.Proofreader,
	queue port.ProofreadQueue,
	v domain.Vault,
) {
	recognised, err := known.Recognised(ctx, string(v.ID), domain.KindBook)
	if err != nil {
		return
	}
	said := r.with.Proofreading
	for _, one := range recognised {
		if ctx.Err() != nil {
			return
		}
		id := proofreadingID(one.Path)
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
			// The recognition is held by another run, and that run is the one
			// whose progress the list carries.
		case res.None, res.Read >= res.Pages:
			// A recognition with nothing left to put right is a recognition
			// nobody is waiting on.
			r.done(id)
		default:
			r.says(task.Task{
				ID: id, Doing: "Proofreading a reading", About: one.Path,
				Done: int64(res.Read), Total: int64(res.Pages),
			}, false)
		}
	}
}
