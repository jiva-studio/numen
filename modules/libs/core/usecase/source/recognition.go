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

// recognitionID is what the nth recognition of this launch is called, wherever
// it is shown. It stands for the whole of that recognition, so what it reports
// again replaces itself, and one recognition dismissed is one recognition
// dismissed.
//
// The list it names into lives as long as the process, so counting is the whole
// of what makes one name distinct. Two recognitions can begin inside one tick
// of a clock and share the name it would give them.
func recognitionID(nth uint64) string { return fmt.Sprintf("recognition-%d", nth) }

// proofreadID is what putting one file's text right is called, wherever it is
// shown. One file is one line, and it replaces itself as the text is put right.
func proofreadID(path string) string { return "proofreading-" + path }

// OpenRecogniser opens what reads a scanned page, when there is one to read.
// It is told how far the fetching of what it needs has got.
type OpenRecogniser func(ctx context.Context, tell func(what string, done, total int64)) (port.Recogniser, func() error, error)

// A RecognitionRuntime is what a page is recognised through on this machine.
type RecognitionRuntime struct {
	// Open recognises a page.
	Open OpenRecogniser
	// Ready says whether opening it would wait for anything to arrive.
	Ready func() bool
	// Prepared says whether the runtime was made before the window. One made
	// after a window reads every page it is given as nothing, so a page it
	// would read is the next opening's to read. A test that recognises through
	// nothing leaves this unset.
	Prepared func() bool
}

// Recognitions is what one installation recognises scanned documents with: what
// recognises a page, what a vault is read and written through, and what puts a
// recognition right afterwards.
type Recognitions struct {
	Readers   port.VaultReaders
	Derived   port.DerivedStores
	Documents port.PageRenderer
	Sources   port.SourceRepository
	Tasks     *task.Tasks

	// Runtime is what a page is recognised through.
	Runtime RecognitionRuntime

	// Models is this machine's models and processor, held by one run at a time.
	// A worker given none holds one of its own, and takes its turn with nobody.
	Models *Lock

	// Proofreading is what a reading is put right with.
	Proofreading ProofreadingConfig
}

// RecognitionWorker recognises scanned documents behind whoever asked.
//
// Nothing here is done inside the question that asked for it. Fetching the
// models is minutes and recognising a book is an hour, and an answer that
// arrives in an hour is a program that has hung. The ask starts the work and
// says so, and how far it has got is put where everything else being done is
// put.
type RecognitionWorker struct {
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
	// idle is called where a run has found the line empty and is about to stop
	// the running, under the lock both it and the running are held by. A test
	// names a document there, at the one instant the two could cross.
	idle func()
	// last is what the recognition before this one was called. A recognition
	// that failed is left in the list under that name, and the next recognition
	// takes it out.
	last string
	// named counts the recognitions that have been named, which is what one of
	// them is called by.
	named uint64
}

// NewRecognitionWorker is the recogniser an installation offers, reporting
// itself into the list of what is being done.
func NewRecognitionWorker(ctx context.Context, with Recognitions) *RecognitionWorker {
	if with.Models == nil {
		with.Models = &Lock{}
	}
	return &RecognitionWorker{with: with, under: ctx}
}

// Ready says whether a document could be recognised now without waiting for
// anything to arrive.
func (r *RecognitionWorker) Ready() bool { return r.with.Runtime.Ready() }

// Wait is every recognition and every collection this started, ended. What they
// write goes into an index the application still holds open.
func (r *RecognitionWorker) Wait() { r.going.Wait() }

// IsRunning says whether a document is being recognised.
func (r *RecognitionWorker) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Start recognises one document a person named, and says whether it began now
// or waits its turn.
//
// One at a time: the models hold a worker each. A document named while one is
// being recognised goes to the back of the line and is recognised as soon as
// the run before it ends.
//
// It runs under the application, so whoever asked is answered at once and goes
// away while the recognition carries on.
func (r *RecognitionWorker) Start(v domain.Vault, path string) port.StartOutcome {
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
		defer r.stopRun(mine)
		r.drain(r.context())
	}()
	return port.Began
}

// CountWaiting is how many documents a person named are still in line.
func (r *RecognitionWorker) CountWaiting() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.queue.countWaiting()
}

// drain recognises every document a person named, in the order they named them.
// A document is out of the line before it is recognised, so one whose
// recognition ends where nothing expected it to holds nothing afterwards.
// The line is found empty and the running stopped under one hold of the lock,
// so a document named at that instant is answered "began" and recognised by the
// run that answers it.
//
// The context is asked before a document is taken, so what the line still holds
// when the application closes is still in it.
func (r *RecognitionWorker) drain(ctx context.Context) {
	for {
		r.mu.Lock()
		var (
			one     wanted
			waiting bool
		)
		if ctx.Err() == nil {
			one, waiting = r.queue.take()
		}
		if !waiting {
			if r.idle != nil {
				r.idle()
			}
			r.running = false
			r.mu.Unlock()
			return
		}
		r.mu.Unlock()
		r.one(ctx, one.vault, one.path)
	}
}

// one is a single document recognised, put right, and reported.
func (r *RecognitionWorker) one(ctx context.Context, v domain.Vault, path string) {
	r.mu.Lock()
	before := r.last
	r.named++
	id := recognitionID(r.named)
	r.last = id
	r.mu.Unlock()

	r.finishTask(before)
	r.say(task.Task{ID: id, Doing: "Reading a scan", About: path})

	err := r.recognise(ctx, v, id, path)
	if err == nil {
		r.proofread(ctx, v, path)
	}

	switch {
	case err == nil, errors.Is(err, context.Canceled):
		// A recognition somebody stopped is a recognition that is over.
		r.finishTask(id)
	default:
		// A failure nobody was shown is a failure nobody can act on, so it
		// stays in the list until it is dismissed or the next recognition
		// begins.
		r.say(task.Task{ID: id, Doing: "Reading a scan", About: path, Error: err.Error()})
	}
}

// stopRun stops the running where this run is still the one running, so a
// recognition that ended where nothing expected it to leaves nothing running.
func (r *RecognitionWorker) stopRun(run uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runs == run {
		r.running = false
	}
}

func (r *RecognitionWorker) context() context.Context {
	if r.under == nil {
		return context.Background()
	}
	return r.under
}

// recognise is the work itself: what is missing arrives, and then the document
// is recognised.
func (r *RecognitionWorker) recognise(ctx context.Context, v domain.Vault, id, path string) (err error) {
	// One run holds the models on a machine: a recording being transcribed holds
	// them, and this waits for it.
	// A scan is recognised only where somebody asked for it.
	release, err := r.with.Models.acquire(ctx, true, func() {
		r.say(task.Task{ID: id, Doing: "Waiting for the models", About: path})
	})
	if err != nil {
		return err
	}
	defer release()

	// Getting the models is a step of its own and stands under its own name.
	// Which file is coming down, and how much of it, is known once one is.
	r.say(task.Task{ID: id, Doing: "Fetching models"})
	by, letGo, err := r.with.Runtime.Open(ctx, func(what string, done, total int64) {
		// The count is bytes and says so, and the sizes a person reads them in
		// are the window's to write.
		r.say(task.Task{
			ID: id, Doing: "Fetching models", About: what,
			Count: done, Total: total, Unit: task.Bytes,
		})
	})
	if err != nil {
		return fmt.Errorf("nothing to recognise with: %w", err)
	}
	// Letting the models go is what leaves the machine able to read again. A
	// machine that will not is worth saying, and the pages read stand.
	defer func() {
		if why := letGo(); why != nil {
			err = errors.Join(err, fmt.Errorf("letting the models go: %w", why))
		}
	}()

	// Every page of this process is recognised through the runtime it made before
	// its window, and a runtime this process fetched afterwards is not that one.
	// What was missing is here now, and the recognition is the next opening's to
	// do.
	if r.with.Runtime.Prepared != nil && !r.with.Runtime.Prepared() {
		return errLateRuntime
	}

	r.say(task.Task{ID: id, Doing: "Reading a scan", About: path})
	read := NewRecognise(r.with.Readers, r.with.Sources, r.with.Derived, r.with.Documents, by)
	read.Cut = r.Cut
	read.OnProgress = func(res RecogniseResult) {
		r.say(task.Task{
			ID:    id,
			Doing: "Reading a scan",
			About: path,
			Count: int64(res.Read),
			Total: int64(res.Pages),
		})
	}
	res, err := read.Execute(ctx, v, path)
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

// proofread puts a reading right, where a person configured something to
// proofread it with. An installation that named no profile, or asked for a
// reading to be put right by hand, does nothing here.
//
// It reports itself under its own name, and a reading whose proofreading
// failed is the reading as it was read.
func (r *RecognitionWorker) proofread(ctx context.Context, v domain.Vault, path string) {
	said := r.with.Proofreading
	if !said.Automatically {
		return
	}

	right, held, err := said.Reading(r.with.Readers, r.with.Derived)
	if err != nil {
		r.say(task.Task{ID: proofreadID(path), Doing: "Proofreading a reading", About: path, Error: err.Error()})
		return
	}
	if !held {
		return
	}

	id := proofreadID(path)
	r.say(task.Task{ID: id, Doing: "Proofreading a reading", About: path})

	right.Cut = r.Cut
	right.OnProgress = func(res ProofreadReadingResult) {
		r.say(task.Task{
			ID:    id,
			Doing: "Proofreading a reading",
			About: path,
			Count: int64(res.Read),
			Total: int64(res.Pages),
		})
	}
	_, err = right.Execute(ctx, v, path)

	switch {
	case err == nil, errors.Is(err, context.Canceled):
		r.finishTask(id)
	default:
		r.say(task.Task{ID: id, Doing: "Proofreading a reading", About: path, Error: err.Error()})
	}
}

// say puts this recognition in the list of what is being done. A person asked
// for it and is waiting to be told it began.
func (r *RecognitionWorker) say(at task.Task) { r.says(at, true) }

// says puts one piece of work in the list. Work a person started is shown at
// once, and work nobody asked for is shown once it has lasted.
func (r *RecognitionWorker) says(at task.Task, asked bool) {
	at.Asked = asked
	if r.with.Tasks != nil {
		r.with.Tasks.Set(at)
	}
}

func (r *RecognitionWorker) finishTask(id string) {
	if r.with.Tasks != nil {
		r.with.Tasks.Remove(id)
	}
}

// Collecting asks after the batches left with a proofreader, until the context
// is done, behind the caller. A batch outlives the run that left it, so one
// left before the application closed is collected when it opens.
//
// Nothing here is done unless a person configured a proofreader with a queue.
//
// The count is taken here and not in the goroutine it counts, so a wait that
// begins the instant this returns covers the rounds behind it.
func (r *RecognitionWorker) CollectBatches(
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
	go func() {
		defer r.going.Done()
		r.runRounds(ctx, known, queue, every, vaults)
	}()
}

// runRounds is the round over every vault, and the wait between rounds.
func (r *RecognitionWorker) runRounds(
	ctx context.Context,
	known port.SourceQueries,
	queue port.ProofreadQueue,
	every time.Duration,
	vaults []domain.Vault,
) {
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

// TakeUp puts right the readings of these vaults that stand short of
// their last page, once, behind the caller.
//
// A proofreading stands at the page it reached, so a run that ended among the
// batches is taken up at that page. A reading no proofreader has been over
// stands at its first page and is put right whole. A proofreader with a queue
// leaves a batch behind it and is taken up by CollectBatches.
func (r *RecognitionWorker) TakeUp(
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
func (r *RecognitionWorker) collect(
	ctx context.Context,
	known port.SourceQueries,
	by port.Proofreader,
	queue port.ProofreadQueue,
	v domain.Vault,
) {
	recognised, err := known.GetRecognisedSources(ctx, v.ID, domain.KindBook)
	if err != nil {
		return
	}
	said := r.with.Proofreading
	for _, one := range recognised {
		if ctx.Err() != nil {
			return
		}
		id := proofreadID(one.Path)
		res, err := ProofreadReading{
			Readers:         r.with.Readers,
			Derived:         r.with.Derived,
			By:              by,
			Queue:           queue,
			Pages:           said.Batch,
			MaxEditDistance: said.MaxEditDistance,
			Cut:             r.Cut,
			OnProgress: func(res ProofreadReadingResult) {
				r.says(task.Task{
					ID: id, Doing: "Proofreading a reading", About: one.Path,
					Count: int64(res.Read), Total: int64(res.Pages),
				}, false)
			},
		}.Execute(ctx, v, one.Path)

		switch {
		case err != nil:
			r.says(task.Task{
				ID: id, Doing: "Proofreading a reading",
				About: one.Path, Error: err.Error(),
			}, false)
		case res.Busy:
			// The reading is held by another run, and that run is the one
			// whose progress the list carries.
		case res.None, res.Read >= res.Pages:
			// A reading with nothing left to put right is a run nobody is
			// waiting on.
			r.finishTask(id)
		default:
			r.says(task.Task{
				ID: id, Doing: "Proofreading a reading", About: one.Path,
				Count: int64(res.Read), Total: int64(res.Pages),
			}, false)
		}
	}
}
