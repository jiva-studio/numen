package transcription

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// errNothingTranscribes is a run that reached nothing: what transcribes a
// recording is not on this machine. It is about the machine and not about the
// file, so a queue leaves the work where it is and comes back to it.
var errNothingTranscribes = errors.New("nothing to transcribe with")

const (
	taskTranscribingRecording  = "Transcribing a recording"
	taskProofreadingTranscript = "Proofreading a transcript"
)

// transcriptionID is what one recording's transcription is called, wherever it
// is shown. One recording is one line, and it replaces itself as the words are
// written down.
func transcriptionID(path string) string { return "transcription-" + path }

// OpenTranscriber opens what listens to a recording, when there is one to
// listen to. It is told how far the fetching of what it needs has got.
type OpenTranscriber func(ctx context.Context, tell func(what string, finished, total int64)) (port.Transcriber, func() error, error)

// A Runtime is what a recording is transcribed through on this
// machine.
type Runtime struct {
	// Open transcribes a recording.
	Open OpenTranscriber
	// Ready says whether opening it would wait for anything to arrive.
	Ready func() bool
	// CanKeep says whether the opened models may stay loaded between
	// recordings. Absent, they are opened and closed for each recording.
	CanKeep func() bool
}

// Config is what one installation transcribes recordings with: what
// transcribes them, what a vault is read and written through, and what puts a
// transcript right afterwards.
type Config struct {
	Readers port.VaultReaders
	Derived port.DerivedStores
	Sources port.SourceRepository
	Tasks   *run.Tasks

	// Runtime is what a recording is transcribed through.
	Runtime Runtime

	// Models is this machine's models and processor, held by one run at a time.
	// A worker given none holds one of its own, and takes its turn with nobody.
	Models *run.Lock

	// Unasked is how many bytes a recording may run to and still be transcribed
	// where nobody asked for it. A larger one is left for somebody to ask for by
	// name. Zero is no limit.
	Unasked int64

	// Proofreading is what a transcript is put right with.
	Proofreading run.ProofreadingConfig

	// Wait holds the queue between its rounds, and Delay closes the parked models
	// once they have been idle.
	Wait  port.Wait
	Delay port.Delay
}

// Worker transcribes recordings behind whoever asked, and behind
// nobody.
//
// A recording carries no text of its own, so a vault of recordings is work to
// be done whether or not anybody asks. The hand asks through Start and the
// queue asks through Queue, and both arrive here.
type Worker struct {
	with Config

	// line runs the recordings a person named, one at a time.
	line *run.Worker

	// Cut makes a source's chunks from what has been transcribed of it. It is
	// called as speech is written down, so the beginning of a recording is
	// searchable while the end of it is still being transcribed.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	mu sync.Mutex
	// answered is every recording this run has had an answer about, by vault
	// and path. Words, silence and a file that will not open are all answers,
	// and a recording that has answered is not offered again.
	processed map[string]bool

	// resident is the opened models kept loaded between the recordings of a run.
	resident resident
	// residentIdle is how long they stay loaded with no recording to hear.
	residentIdle time.Duration
}

// defaultResidentIdle is how long opened models stay loaded with no recording
// to hear; loaded models hold memory other work may need.
const defaultResidentIdle = 30 * time.Second

// NewWorker is the transcriber an installation offers, reporting
// itself into the list of what is being done.
func NewWorker(ctx context.Context, with Config) *Worker {
	if with.Models == nil {
		with.Models = &run.Lock{}
	}
	t := &Worker{with: with, processed: map[string]bool{}, residentIdle: defaultResidentIdle}
	t.line = run.NewWorker(ctx, with.Tasks, t.transcribeNamed)
	return t
}

// Ready says whether a recording could be transcribed now without waiting for
// anything to arrive.
func (t *Worker) Ready() bool { return t.with.Runtime.Ready() }

// Wait is every transcription this started, ended. What they write goes into an
// index the application still holds open.
func (t *Worker) Wait() { t.line.Wait() }

// Start transcribes one recording a person named, and says whether it began now
// or waits its turn.
//
// One at a time: the models hold a worker each. A recording named while one is
// being transcribed goes to the back of the line and is transcribed as soon as
// the run before it ends, ahead of everything the vault set itself.
func (t *Worker) Start(v domain.Vault, path string) port.StartOutcome {
	return t.line.Start(v, path)
}

// CountWaiting is how many recordings a person named are still in line.
func (t *Worker) CountWaiting() int { return t.line.CountWaiting() }

// transcribeNamed works a recording a person named. One that ends in nothing
// has already said so under its own line.
func (t *Worker) transcribeNamed(ctx context.Context, v domain.Vault, path string) {
	_ = t.transcribeOne(ctx, v, path, true)
}

// Queue transcribes every recording this vault holds no transcript for, one
// after another, until the context ends, behind the caller.
//
// The queue is not stored: which recordings owe their text is a question the
// index already answers, so a round interrupted by the application closing is
// taken up when it opens, and it is asked again every so often.
//
// The count is taken here and not in the goroutine it counts, so a wait that
// begins the instant this returns covers the rounds behind it.
func (t *Worker) Queue(
	ctx context.Context,
	queries port.SourceQueries,
	every time.Duration,
	v domain.Vault,
) {
	t.line.Go(func() {
		t.line.RepeatRounds(ctx, t.with.Wait, every, func() { t.round(ctx, queries, v) })
	})
}

// round hands over the recordings a person named and then the ones that owe
// their text, one after another. It ends early where what stopped a recording
// was the machine and not the file: the rest of the round would reach the same
// nothing.
func (t *Worker) round(ctx context.Context, indexed port.SourceQueries, v domain.Vault) {
	mine, free := t.line.Claim()
	if !free {
		// The hand is transcribing something. What this round did not hand over
		// is handed over by the next one.
		return
	}
	defer t.line.Release(mine)

	t.line.Drain(ctx)
	for _, path := range t.getUntranscribed(ctx, indexed, v) {
		if ctx.Err() != nil {
			break
		}
		err := t.transcribeOne(ctx, v, path, false)
		if errors.Is(err, errNothingTranscribes) {
			break
		}
		// A recording named while this round ran is transcribed before the next
		// one the vault owes.
		t.line.Drain(ctx)
	}
	t.line.DrainAndRelease(ctx, t.closeResident)
}

// getUntranscribed is the recordings this vault holds that no model has written
// the words of, and that this run has had no answer about.
//
// The index knows both: every recording it holds, and which of them stand on a
// text a producer made. What is left of the first by the second is the work.
func (t *Worker) getUntranscribed(ctx context.Context, existing port.SourceQueries, v domain.Vault) []string {
	if existing == nil {
		return nil
	}
	target, err := existing.GetFingerprints(ctx, v.ID, domain.KindRecording)
	if err != nil {
		return nil
	}
	recognised, err := existing.GetRecognisedSources(ctx, v.ID, domain.KindRecording)
	if err != nil {
		return nil
	}
	for _, one := range recognised {
		delete(target, one.Path)
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	underVal := t.with.Unasked
	out := make([]string, 0, len(target))
	for path, ref := range target {
		if t.processed[run.GetKey(v, path)] {
			continue
		}
		if underVal > 0 && ref.Size > underVal {
			// A recording this large is somebody's music or somebody's archive.
			// It is transcribed when it is asked for by name.
			continue
		}
		out = append(out, path)
	}
	slices.Sort(out)
	return out
}

// closeResident lets the kept models go when a run ends. A close that fails
// leaves nothing to retry.
func (t *Worker) closeResident() { _ = t.resident.close() }

// transcribeOne is a single recording transcribed, put right, and reported.
//
// isUserStarted says whether a person is waiting to be told it began. Whatever it ends
// as is recorded, and a recording that ended in an answer is not offered again:
// words, silence and a file nothing here can open are all answers, and only
// bytes somebody else holds mean come back later.
func (t *Worker) transcribeOne(ctx context.Context, v domain.Vault, path string, isUserStarted bool) error {
	id := transcriptionID(path)
	t.reportTask(run.Task{ID: id, Title: taskTranscribingRecording, About: path}, isUserStarted)

	res, err := t.transcribe(ctx, v, id, path, isUserStarted)
	switch {
	case errors.Is(err, context.Canceled):
		// A transcription somebody stopped is one that is over, and the
		// recording is where the next round finds it.
		t.finishTask(id)
	case err != nil:
		// The recording is left where the next round finds it. A store that
		// would not write and an index that would not answer are the machine,
		// and the recording has said nothing about itself.
		t.reportTask(run.Task{ID: id, Title: taskTranscribingRecording, About: path, Error: err.Error()}, isUserStarted)
	case res.IsBusy:
		t.finishTask(id)
	default:
		t.finishTask(id)
		t.recordAnswer(v, path)
		if !res.IsSilent && !res.IsUnopened {
			t.proofread(ctx, v, path, "", isUserStarted)
		}
	}
	return err
}

// ProofreaderReady says whether this installation has anything to put a
// transcript right with. A person is offered the run where it has.
func (t *Worker) ProofreaderReady() bool { return t.with.Proofreading.IsNamed }

// what came of asking: the run began, or the transcript needed nothing of it.
//
// The asking waits only for what a run settles before it puts a question to the
// proofreader, which is a few reads off the disk. The work itself carries on
// behind the answer, under the line in the list of what is being done that a
// proofreading of that recording carries.
func (t *Worker) Proofread(
	ctx context.Context,
	v domain.Vault,
	path string,
) (ProofreadResult, error) {
	// A run says two things at most: that it began, and how it ended. The room
	// for both is here, so a run whose caller has gone says them and ends.
	outcomes := make(chan outcome, 2)

	//nolint:contextcheck // the run outlives the caller and carries t.line.Context()
	t.line.Go(func() {
		res, err := t.proofreadTranscript(t.line.Context(), v, path, "", true, func(began ProofreadResult) {
			outcomes <- outcome{res: began}
		})
		outcomes <- outcome{res: res, err: err}
	})

	select {
	case one := <-outcomes:
		return one.res, one.err
	case <-ctx.Done():
		return ProofreadResult{Path: path}, ctx.Err()
	}
}

// outcome is what a run putting a transcript right says about itself: what it
// found, and what stopped it.
type outcome struct {
	res ProofreadResult
	err error
}

// proofread puts a transcript right, where an installation asked for its
// transcripts to be put right on their own.
func (t *Worker) proofread(ctx context.Context, v domain.Vault, path, hash string, isUserStarted bool) {
	if !t.with.Proofreading.IsAutomatic {
		return
	}
	_, _ = t.proofreadTranscript(ctx, v, path, hash, isUserStarted, nil)
}

// proofreadTranscript puts one transcript right with the profile named for
// speech.
//
// A transcript whose proofreading failed is the transcript as it was
// somebody reads it. Work a person asked for is in that list from the moment
// they asked.
func (t *Worker) proofreadTranscript(
	ctx context.Context,
	v domain.Vault,
	path, hash string,
	isUserStarted bool,
	began func(ProofreadResult),
) (ProofreadResult, error) {
	proofreading := t.with.Proofreading

	id := run.GetProofreadID(path)
	fail := func(err error) {
		t.reportTask(run.Task{
			ID: id, Title: taskProofreadingTranscript, About: path, Error: run.GetProofreadError(err),
		}, isUserStarted)
	}
	if isUserStarted {
		t.reportTask(run.Task{ID: id, Title: taskProofreadingTranscript, About: path}, isUserStarted)
	}

	right, target, err := CreateProofread(proofreading, t.with.Readers, t.with.Derived)
	if err != nil {
		fail(err)
		return ProofreadResult{Path: path}, err
	}
	if !target {
		t.finishTask(id)
		return ProofreadResult{Path: path}, nil
	}

	right.Hash = hash
	var once sync.Once
	right.Cut = t.Cut
	right.OnProgress = func(res ProofreadResult) {
		// Progress is reported once there is a question to put, so the first of
		// it is this run beginning.
		once.Do(func() {
			if began != nil {
				began(res)
			}
		})
		t.reportTask(run.Task{
			ID:    id,
			Title: taskProofreadingTranscript,
			About: path,
			Count: int64(res.Read),
			Total: int64(res.Lines),
		}, isUserStarted)
	}
	res, err := right.Execute(ctx, v, path)

	switch {
	case err != nil && !errors.Is(err, context.Canceled):
		fail(err)
	case res.IsBusy && !isUserStarted:
		// The transcript is held by another run, and that run is the one whose
		// progress the list carries.
	default:
		t.finishTask(id)
	}
	return res, err
}

// TakeUp puts right the transcripts of these vaults that stand short of their
// last line, once, behind the caller.
//
// batches is taken up at that line. A transcript no proofreader has been over
// holds is a question the index already answers.
func (t *Worker) TakeUp(
	ctx context.Context,
	sources port.SourceQueries,
	vaults ...domain.Vault,
) {
	if !t.with.Proofreading.IsAutomatic || sources == nil {
		return
	}
	t.line.Go(func() {
		for _, v := range vaults {
			recognised, err := sources.GetRecognisedSources(ctx, v.ID, domain.KindRecording)
			if err != nil {
				continue
			}
			for _, sourceItem := range recognised {
				if ctx.Err() != nil {
					return
				}
				t.proofread(ctx, v, sourceItem.Path, sourceItem.Hash, false)
			}
		}
	})
}

// transcribe is the work itself: this machine's models held, what is missing
// arriving, and then the recording transcribed.
func (t *Worker) transcribe(
	ctx context.Context,
	v domain.Vault,
	id, path string,
	isUserStarted bool,
) (res TranscribeResult, err error) {
	delta, finish, err := t.acquireModels(ctx, id, path, isUserStarted)
	if err != nil {
		return res, err
	}
	defer func() { err = errors.Join(err, finish(err)) }()

	t.reportTask(run.Task{ID: id, Title: taskTranscribingRecording, About: path}, isUserStarted)
	listen := NewTranscribe(t.with.Readers, t.with.Sources, t.with.Derived, delta)
	listen.Cut = t.Cut
	listen.OnProgress = func(res TranscribeResult) {
		t.reportTask(run.Task{
			ID:    id,
			Title: taskTranscribingRecording,
			About: path,
			Count: int64(res.Heard / 1000),
			Total: int64(res.Length / 1000),
			Unit:  run.Seconds,
		}, isUserStarted)
	}
	return listen.Execute(ctx, v, path)
}

// acquireModels hands over the models for one recording, and finish says what
// became of it. A scan being recognised holds the models, and this waits for it.
//
// Models that may stay loaded stay with the run, held under the lock, until
// someone else waits for the lock, the run fails or is cancelled, no recording
// follows within the idle period, or the run ends.
func (t *Worker) acquireModels(
	ctx context.Context,
	id, path string,
	isUserStarted bool,
) (delta port.Transcriber, finish func(ran error) error, err error) {
	if retained, ok := t.resident.take(); ok {
		return retained, func(ran error) error { return t.finishResident(ctx, ran) }, nil
	}

	release, err := t.with.Models.Acquire(ctx, isUserStarted, func() {
		t.reportTask(run.Task{ID: id, Title: "Waiting for the models", About: path}, isUserStarted)
	})
	if err != nil {
		return nil, nil, err
	}

	// Getting the models is a step of its own and is placed under its own name.
	// Which file is coming down, and how much of it, is known once one is.
	t.reportTask(run.Task{ID: id, Title: "Fetching models"}, isUserStarted)
	delta, letGo, err := t.with.Runtime.Open(ctx, func(what string, completed, total int64) {
		// The count is bytes and says so, and the sizes a person reads them in
		// are the window's to write.
		t.reportTask(run.Task{
			ID: id, Title: "Fetching models", About: what,
			Count: completed, Total: total, Unit: run.Bytes,
		}, isUserStarted)
	})
	if err != nil {
		release()
		return nil, nil, fmt.Errorf("%w: %w", errNothingTranscribes, err)
	}

	if t.with.Runtime.CanKeep != nil && t.with.Runtime.CanKeep() {
		t.resident.hold(delta, letGo, release)
		return delta, func(ran error) error { return t.finishResident(ctx, ran) }, nil
	}
	// Letting the models go is what leaves the machine able to transcribe again.
	// A machine that will not is worth saying, and the transcript stands.
	return delta, func(error) error {
		defer release()
		if why := letGo(); why != nil {
			return fmt.Errorf("letting the models go: %w", why)
		}
		return nil
	}, nil
}

// finishResident keeps the models loaded after a recording that succeeded while
// nobody waits for them, and lets them go otherwise.
func (t *Worker) finishResident(ctx context.Context, ran error) error {
	if ran != nil || ctx.Err() != nil || t.with.Models.HasWaiting() {
		if why := t.resident.close(); why != nil {
			return fmt.Errorf("letting the models go: %w", why)
		}
		return nil
	}
	t.resident.park(t.with.Delay, t.residentIdle)
	return nil
}

// recordAnswer puts a recording out of the queue's reach for the life of this
// run. The answer itself is a file in the vault, so the next run reads it back
// and hands the recording over once and no more.
func (t *Worker) recordAnswer(v domain.Vault, path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.processed[run.GetKey(v, path)] = true
}

// ClearAnswer puts a recording back within the queue's reach. The answer it gave is
// gone from the vault, and the queue hands it over again.
func (t *Worker) ClearAnswer(v domain.Vault, path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.processed, run.GetKey(v, path))
}

// reportTask puts this transcription in the list of what is being done. Work nobody
// asked for is shown once it has lasted, and work a person started is shown at
// once.
func (t *Worker) reportTask(pos run.Task, isUserStarted bool) {
	t.line.Report(pos, isUserStarted)
}

func (t *Worker) finishTask(id string) { t.line.Finish(id) }
