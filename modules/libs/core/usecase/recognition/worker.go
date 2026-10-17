package recognition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

// errLateRuntime is a document left unrecognised because what would recognise
// it was fetched after the window was made.
var errLateRuntime = errors.New("what reads a scan arrived just now; open numen again to read it")

// recognitionID is what one document's recognition is called, wherever it is
// shown. One document is one line, and it replaces itself as the pages are
// read.
func recognitionID(path string) string { return "recognition-" + path }

// OpenRecogniser opens what reads a scanned page, when there is one to read.
// It is told how far the fetching of what it needs has got.
type OpenRecogniser func(ctx context.Context, tell func(what string, done, total int64)) (port.Recogniser, func() error, error)

// A Runtime is what a page is recognised through on this machine.
type Runtime struct {
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

// Config is what one installation recognises scanned documents with: what
// recognises a page, what a vault is read and written through, and what puts a
// recognition right afterwards.
type Config struct {
	Readers   port.VaultReaders
	Derived   port.DerivedStores
	Documents port.PageRenderer
	Sources   port.SourceRepository
	Tasks     *run.Tasks

	// Runtime is what a page is recognised through.
	Runtime Runtime

	// Models is this machine's models and processor, held by one run at a time.
	// A worker given none holds one of its own, and takes its turn with nobody.
	Models *run.Lock

	// Proofreading is what a recognition is put right with.
	Proofreading run.ProofreadingConfig

	// Wait holds the collection of batches between its rounds.
	Wait port.Wait
}

// Worker recognises scanned documents behind whoever asked.
//
// Nothing here is done inside the question that asked for it. Fetching the
// models is minutes and recognising a book is an hour, and an answer that
// arrives in an hour is a program that has hung. The ask starts the work and
// says so, and how far it has got is put where everything else being done is
// put.
type Worker struct {
	with Config

	// line runs the documents a person named, one at a time.
	line *run.Worker

	// Cut makes a source's chunks from what has been recognised of it. It is
	// called as pages are written down, so a page is searchable when it is
	// recognised.
	Cut func(ctx context.Context, v domain.Vault, path string) error
}

// NewWorker is the recogniser an installation offers, reporting
// itself into the list of what is being done.
func NewWorker(ctx context.Context, with Config) *Worker {
	if with.Models == nil {
		with.Models = &run.Lock{}
	}
	r := &Worker{with: with}
	r.line = run.NewWorker(ctx, with.Tasks, r.recogniseOne)
	return r
}

// Ready says whether a document could be recognised now without waiting for
// anything to arrive.
func (r *Worker) Ready() bool { return r.with.Runtime.Ready() }

// Wait is every recognition and every collection this started, ended. What they
// write goes into an index the application still holds open.
func (r *Worker) Wait() { r.line.Wait() }

// IsRunning says whether a document is being recognised.
func (r *Worker) IsRunning() bool { return r.line.IsRunning() }

// Start recognises one document a person named, and says whether it began now
// or waits its turn.
//
// One at a time: the models hold a worker each. A document named while one is
// being recognised goes to the back of the line and is recognised as soon as
// the run before it ends.
func (r *Worker) Start(v domain.Vault, path string) port.StartOutcome {
	return r.line.Start(v, path)
}

// CountWaiting is how many documents a person named are still in line.
func (r *Worker) CountWaiting() int { return r.line.CountWaiting() }

// recogniseOne is a single document recognised, put right, and reported.
func (r *Worker) recogniseOne(ctx context.Context, v domain.Vault, path string) {
	id := recognitionID(path)
	r.reportTask(run.Task{ID: id, Title: "Recognising a scan", About: path})

	err := r.recognise(ctx, v, id, path)
	if err == nil {
		r.proofread(ctx, v, path)
	}

	switch {
	case err == nil, errors.Is(err, context.Canceled):
		// A recognition somebody stopped is a recognition that is over.
		r.finishTask(id)
	default:
		// A failure stands in the list until whoever is shown it takes it out.
		r.reportTask(run.Task{ID: id, Title: "Recognising a scan", About: path, Error: err.Error()})
	}
}

// recognise is the work itself: what is missing arrives, and then the document
// is recognised.
func (r *Worker) recognise(ctx context.Context, v domain.Vault, id, path string) (err error) {
	// One run holds the models on a machine: a recording being transcribed holds
	// them, and this waits for it.
	// A scan is recognised only where somebody asked for it.
	release, err := r.with.Models.Acquire(ctx, true, func() {
		r.reportTask(run.Task{ID: id, Title: "Waiting for the models", About: path})
	})
	if err != nil {
		return err
	}
	defer release()

	// Getting the models is a step of its own and stands under its own name.
	// Which file is coming down, and how much of it, is known once one is.
	r.reportTask(run.Task{ID: id, Title: "Fetching models"})
	by, letGo, err := r.with.Runtime.Open(ctx, func(what string, done, total int64) {
		// The count is bytes and says so, and the sizes a person reads them in
		// are the window's to write.
		r.reportTask(run.Task{
			ID: id, Title: "Fetching models", About: what,
			Count: done, Total: total, Unit: run.Bytes,
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

	r.reportTask(run.Task{ID: id, Title: "Recognising a scan", About: path})
	read := NewRecognise(r.with.Readers, r.with.Sources, r.with.Derived, r.with.Documents, by)
	read.Cut = r.Cut
	read.OnProgress = func(res RecogniseResult) {
		r.reportTask(run.Task{
			ID:    id,
			Title: "Recognising a scan",
			About: path,
			Count: int64(res.Read),
			Total: int64(res.Pages),
		})
	}
	res, err := read.Execute(ctx, v, path)
	if err != nil {
		return err
	}
	if res.IsBusy {
		// Another run holds these bytes — a terminal, or a second window. What
		// it recognises is what this recognises.
		return fmt.Errorf("%s is already being read", path)
	}
	return nil
}

// ProofreaderReady says whether this installation has anything to put a
// recognition right with. A person is offered the run where it has.
func (r *Worker) ProofreaderReady() bool { return r.with.Proofreading.IsNamed }

// Proofread puts one recognition right for a person who asked for it and says
// what came of asking: the run began, or the recognition needed nothing of it.
func (r *Worker) Proofread(
	ctx context.Context,
	v domain.Vault,
	path string,
) (ProofreadResult, error) {
	outcomes := make(chan proofreadOutcome, 2)
	//nolint:contextcheck // the run outlives the caller and carries r.line.Context()
	r.line.Go(func() {
		res, err := r.proofreadRecognition(r.line.Context(), v, path, func(began ProofreadResult) {
			outcomes <- proofreadOutcome{res: began}
		})
		outcomes <- proofreadOutcome{res: res, err: err}
	})

	select {
	case one := <-outcomes:
		return one.res, one.err
	case <-ctx.Done():
		return ProofreadResult{Path: path}, ctx.Err()
	}
}

type proofreadOutcome struct {
	res ProofreadResult
	err error
}

// proofread puts a recognition right, where a person configured something to
// proofread it with. An installation that named no profile, or asked for a
// recognition to be put right by hand, does nothing here.
func (r *Worker) proofread(ctx context.Context, v domain.Vault, path string) {
	if !r.with.Proofreading.IsAutomatic {
		return
	}
	_, _ = r.proofreadRecognition(ctx, v, path, nil)
}

func (r *Worker) proofreadRecognition(
	ctx context.Context,
	v domain.Vault,
	path string,
	onBegan func(ProofreadResult),
) (ProofreadResult, error) {
	proofreading := r.with.Proofreading
	right, held, err := CreateProofread(proofreading, r.with.Readers, r.with.Derived)
	if err != nil {
		r.reportTask(run.Task{ID: run.GetProofreadID(path), Title: "Proofreading a recognition", About: path, Error: err.Error()})
		return ProofreadResult{Path: path}, err
	}
	if !held {
		return ProofreadResult{Path: path, IsNone: true}, nil
	}

	id := run.GetProofreadID(path)
	r.reportTask(run.Task{ID: id, Title: "Proofreading a recognition", About: path})

	right.Cut = r.Cut
	right.Pages = proofreading.Batch
	right.MaxEditDistance = proofreading.MaxEditDistance
	right.OnProgress = func(res ProofreadResult) {
		r.reportTask(run.Task{
			ID:    id,
			Title: "Proofreading a recognition",
			About: path,
			Count: int64(res.Read),
			Total: int64(res.Pages),
		})
		if onBegan != nil {
			onBegan(res)
			onBegan = nil
		}
	}
	res, err := right.Execute(ctx, v, path)

	switch {
	case err == nil, errors.Is(err, context.Canceled):
		r.finishTask(id)
	default:
		r.reportTask(run.Task{ID: id, Title: "Proofreading a recognition", About: path, Error: err.Error()})
	}
	return res, err
}

// reportTask puts this recognition in the list of what is being done, shown at
// once because a person asked for it.
func (r *Worker) reportTask(at run.Task) { r.line.Report(at, true) }

func (r *Worker) finishTask(id string) { r.line.Finish(id) }

// CollectBatches asks after the batches left with a proofreader, until the
// context is done, behind the caller. A batch outlives the run that left it, so
// one left before the application closed is collected when it opens.
//
// Nothing here is done unless a person configured a proofreader with a queue.
//
// The count is taken here and not in the goroutine it counts, so a wait that
// begins the instant this returns covers the rounds behind it.
func (r *Worker) CollectBatches(
	ctx context.Context,
	known port.SourceQueries,
	every time.Duration,
	vaults ...domain.Vault,
) {
	queue, err := r.with.Proofreading.Queue()
	if err != nil || queue == nil {
		return
	}
	r.line.Go(func() {
		r.line.RepeatRounds(ctx, r.with.Wait, every, func() {
			for _, v := range vaults {
				r.collect(ctx, known, queue, queue, v)
			}
		})
	})
}

// TakeUp puts right the recognitions of these vaults that stand short of
// their last page, once, behind the caller.
//
// A proofreading stands at the page it reached, so a run that ended among the
// batches is taken up at that page. A recognition no proofreader has been over
// stands at its first page and is put right whole. A proofreader with a queue
// leaves a batch behind it and is taken up by CollectBatches.
func (r *Worker) TakeUp(
	ctx context.Context,
	known port.SourceQueries,
	vaults ...domain.Vault,
) {
	proofreading := r.with.Proofreading
	if !proofreading.IsAutomatic {
		return
	}
	r.line.Go(func() {
		queue, err := proofreading.Queue()
		if err != nil || queue != nil {
			return
		}
		by, err := proofreading.By()
		if err != nil || by == nil {
			return
		}
		for _, v := range vaults {
			r.collect(ctx, known, by, nil, v)
		}
	})
}

// collect takes up every recognition of one vault that stands short of its last
// page.
func (r *Worker) collect(
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
	proofreading := r.with.Proofreading
	for _, one := range recognised {
		if ctx.Err() != nil {
			return
		}
		id := run.GetProofreadID(one.Path)
		right, err := NewProofread(r.with.Readers, r.with.Derived, by)
		if err != nil {
			r.line.Report(run.Task{
				ID: id, Title: "Proofreading a recognition",
				About: one.Path, Error: err.Error(),
			}, true)
			continue
		}
		right.Hash = one.Hash
		right.Queue = queue
		right.Pages = proofreading.Batch
		right.MaxEditDistance = proofreading.MaxEditDistance
		right.Cut = r.Cut
		right.OnProgress = func(res ProofreadResult) {
			r.line.Report(run.Task{
				ID: id, Title: "Proofreading a recognition", About: one.Path,
				Count: int64(res.Read), Total: int64(res.Pages),
			}, false)
		}
		res, err := right.Execute(ctx, v, one.Path)

		switch {
		case err != nil:
			r.line.Report(run.Task{
				ID: id, Title: "Proofreading a recognition",
				About: one.Path, Error: err.Error(),
			}, false)
		case res.IsBusy:
			// The recognition is held by another run, and that run is the one
			// whose progress the list carries.
		case res.IsNone, res.Read >= res.Pages:
			// A recognition with nothing left to put right is a run nobody is
			// waiting on.
			r.finishTask(id)
		default:
			r.line.Report(run.Task{
				ID: id, Title: "Proofreading a recognition", About: one.Path,
				Count: int64(res.Read), Total: int64(res.Pages),
			}, false)
		}
	}
}
