package source

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// errNothingTranscribes is a run that reached nothing: what transcribes a
// recording is not on this machine. It is about the machine and not about the
// file, so a queue leaves the work where it is and comes back to it.
var errNothingTranscribes = errors.New("nothing to transcribe with")

// transcriptionID is what one recording's transcription is called, wherever it
// is shown. One recording is one line, and it replaces itself as the words are
// written down.
func transcriptionID(path string) string { return "transcription-" + path }

// OpenTranscriber opens what listens to a recording, when there is one to
// listen to. It is told how far the fetching of what it needs has got.
type OpenTranscriber func(ctx context.Context, tell func(what string, done, total int64)) (port.Transcriber, func() error, error)

// ProofreadingConfig is what a text is put right with: what answers, where
// batches are left for it to answer about later, and how much of a text goes
// over at a time.
//
// Naming no proofreader is a text used exactly as it was made.
type ProofreadingConfig struct {
	// Named says whether a profile is named for this kind of text. A person is
	// offered the run where one is.
	Named bool
	// Automatically says whether a text is put right without anybody asking.
	Automatically bool
	// By opens what answers about a batch, and Queue where batches are left for
	// it to answer about later. Each is told what it is proofreading. Both
	// answer nothing where no profile is named, and the reason where one cannot
	// be opened.
	By    func(instruction string) (port.Proofreader, error)
	Queue func(instruction string) (port.ProofreadQueue, error)
	// Batch is how much of a text one request carries, Overlap how much of it
	// the request before also carried, and InFlight how many stand out at once.
	Batch, Overlap, InFlight int
	// MaxEditDistance is how far a reply may stand from the text before it is
	// refused as an answer about something else.
	MaxEditDistance float64
}

// A TranscriptionRuntime is what a recording is transcribed through on this
// machine.
type TranscriptionRuntime struct {
	// Open transcribes a recording.
	Open OpenTranscriber
	// Ready says whether opening it would wait for anything to arrive.
	Ready func() bool
}

// Transcriptions is what one installation transcribes recordings with: what
// transcribes them, what a vault is read and written through, and what puts a
// transcript right afterwards.
type Transcriptions struct {
	Readers port.VaultReaders
	Derived port.DerivedStores
	Sources port.SourceRepository
	Tasks   *task.Tasks

	// Runtime is what a recording is transcribed through.
	Runtime TranscriptionRuntime

	// Unasked is how many bytes a recording may run to and still be transcribed
	// where nobody asked for it. A larger one is left for somebody to ask for by
	// name. Zero is no limit.
	Unasked int64

	// Proofreading is what a transcript is put right with.
	Proofreading ProofreadingConfig
}

// TranscriptionWorker transcribes recordings behind whoever asked, and behind
// nobody.
//
// A recording carries no text of its own, so a vault of recordings is work to
// be done whether or not anybody asks. The hand asks through Start and the
// queue asks through Queue, and both arrive here.
type TranscriptionWorker struct {
	with Transcriptions

	// under is what every transcription runs under, and going is every
	// transcription that has not ended. One outlives the question that asked
	// for it and ends with the application, which waits here for what it is
	// still writing.
	under context.Context
	going sync.WaitGroup

	// Cut makes a source's chunks from what has been transcribed of it. It is
	// called as speech is written down, so the beginning of a recording is
	// searchable while the end of it is still being transcribed.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	mu      sync.Mutex
	running bool
	// runs counts the runs that have begun. A run stops the running only while
	// it is still the one running.
	runs uint64
	// queue is the recordings a person named that have not been transcribed yet.
	queue queue
	// idle is called where a run has found the line empty and is about to stop
	// the running, under the lock both it and the running are held by. A test
	// names a recording there, at the one instant the two could cross.
	idle func()
	// answered is every recording this run has had an answer about, by vault
	// and path. Words, silence and a file that will not open are all answers,
	// and a recording that has answered is not offered again.
	answered map[string]bool
}

// NewTranscriptionWorker is the transcriber an installation offers, reporting
// itself into the list of what is being done.
func NewTranscriptionWorker(ctx context.Context, with Transcriptions) *TranscriptionWorker {
	return &TranscriptionWorker{with: with, under: ctx, answered: map[string]bool{}}
}

// Ready says whether a recording could be transcribed now without waiting for
// anything to arrive.
func (t *TranscriptionWorker) Ready() bool { return t.with.Runtime.Ready() }

// Wait is every transcription this started, ended. What they write goes into an
// index the application still holds open.
func (t *TranscriptionWorker) Wait() { t.going.Wait() }

// Start transcribes one recording a person named, and says whether it began now
// or waits its turn.
//
// One at a time: the models hold a worker each. A recording named while one is
// being transcribed goes to the back of the line and is transcribed as soon as
// the run before it ends, ahead of everything the vault set itself.
//
// It runs under the application, so whoever asked is answered at once and goes
// away while the transcription carries on.
func (t *TranscriptionWorker) Start(v domain.Vault, path string) port.StartOutcome {
	t.mu.Lock()
	t.queue.add(v, path)
	if t.running {
		t.mu.Unlock()
		return port.Queued
	}
	t.running = true
	t.runs++
	mine := t.runs
	t.mu.Unlock()

	t.going.Add(1)
	go func() {
		defer t.going.Done()
		defer t.release(mine)
		t.drainAndRelease(t.context())
	}()
	return port.Began
}

// Waiting is how many recordings a person named are still in line.
func (t *TranscriptionWorker) Waiting() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.queue.waiting()
}

// drain transcribes every recording a person named, in the order they named
// them. A recording is out of the line before it is transcribed, so one whose
// transcription ends where nothing expected it to holds nothing afterwards.
//
// The context is asked before a recording is taken, so what the line still holds
// when the application closes is still in it: a recording taken and then dropped
// is one nobody is told about, on a count that fell for nothing.
func (t *TranscriptionWorker) drain(ctx context.Context) {
	for {
		t.mu.Lock()
		var (
			one     wanted
			waiting bool
		)
		if ctx.Err() == nil {
			one, waiting = t.queue.take()
		}
		t.mu.Unlock()
		if !waiting {
			return
		}
		// Every recording a person named is worked, and one that ends in
		// nothing has already said so under its own line.
		_ = t.one(ctx, one.vault, one.path, true)
	}
}

// Queue transcribes every recording this vault holds no transcript for, one
// after another, until the context ends, behind the caller.
//
// The queue is not stored: which recordings owe their text is a question the
// index already answers, so a round interrupted by the application closing is
// taken up when it opens. It is asked again every so often, because a recording
// dropped into a folder is one the last round did not see.
//
// The count is taken here and not in the goroutine it counts, so a wait that
// begins the instant this returns covers the rounds behind it.
func (t *TranscriptionWorker) Queue(
	ctx context.Context,
	known port.SourceQueries,
	every time.Duration,
	v domain.Vault,
) {
	t.going.Add(1)
	go func() {
		defer t.going.Done()
		t.queueing(ctx, known, every, v)
	}()
}

// queueing is the round, and the wait between rounds.
func (t *TranscriptionWorker) queueing(
	ctx context.Context,
	known port.SourceQueries,
	every time.Duration,
	v domain.Vault,
) {
	for {
		t.round(ctx, known, v)
		select {
		case <-ctx.Done():
			return
		case <-time.After(every):
		}
	}
}

// round hands over the recordings a person named and then the ones that owe
// their text, one after another. It ends early where what stopped a recording
// was the machine and not the file: the rest of the round would reach the same
// nothing.
func (t *TranscriptionWorker) round(ctx context.Context, known port.SourceQueries, v domain.Vault) {
	mine, free := t.claim()
	if !free {
		// The hand is transcribing something. What this round did not hand over
		// is handed over by the next one.
		return
	}
	defer t.release(mine)

	t.drain(ctx)
	for _, path := range t.owing(ctx, known, v) {
		if ctx.Err() != nil {
			break
		}
		err := t.one(ctx, v, path, false)
		if errors.Is(err, errNothingTranscribes) {
			break
		}
		// A recording named while this round ran is transcribed before the next
		// one the vault owes.
		t.drain(ctx)
	}
	t.drainAndRelease(ctx)
}

// owing is the recordings this vault holds that no model has written the words
// of, and that this run has had no answer about.
//
// The index knows both: every recording it holds, and which of them stand on a
// text a producer made. What is left of the first by the second is the work.
func (t *TranscriptionWorker) owing(ctx context.Context, known port.SourceQueries, v domain.Vault) []string {
	if known == nil {
		return nil
	}
	held, err := known.Fingerprints(ctx, v.ID, domain.KindRecording)
	if err != nil {
		return nil
	}
	recognised, err := known.Recognised(ctx, v.ID, domain.KindRecording)
	if err != nil {
		return nil
	}
	for _, said := range recognised {
		delete(held, said.Path)
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	under := t.with.Unasked
	out := make([]string, 0, len(held))
	for path, ref := range held {
		if t.answered[named(v, path)] {
			continue
		}
		if under > 0 && ref.Size > under {
			// A recording this large is somebody's music or somebody's archive.
			// It is transcribed when it is asked for by name.
			continue
		}
		out = append(out, path)
	}
	slices.Sort(out)
	return out
}

// claim takes this installation's one transcription, and says whether it was
// free.
func (t *TranscriptionWorker) claim() (uint64, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running {
		return 0, false
	}
	t.running = true
	t.runs++
	return t.runs, true
}

// release stops the running where this run is still the one running, so a round
// that ended where nothing expected it to leaves nothing running.
func (t *TranscriptionWorker) release(run uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.runs == run {
		t.running = false
	}
}

// drainAndRelease transcribes everything a person named and stops the running.
// The line is found empty and the running stopped under one hold of the lock,
// so a recording named at that instant is answered "began" and transcribed by
// the run that answers it.
func (t *TranscriptionWorker) drainAndRelease(ctx context.Context) {
	for {
		t.drain(ctx)
		t.mu.Lock()
		if t.queue.waiting() == 0 || ctx.Err() != nil {
			if t.idle != nil {
				t.idle()
			}
			t.running = false
			t.mu.Unlock()
			return
		}
		t.mu.Unlock()
	}
}

// one is a single recording transcribed, put right, and reported.
//
// asked says whether a person is waiting to be told it began. Whatever it ends
// as is recorded, and a recording that ended in an answer is not offered again:
// words, silence and a file nothing here can open are all answers, and only
// bytes somebody else holds mean come back later.
func (t *TranscriptionWorker) one(ctx context.Context, v domain.Vault, path string, asked bool) error {
	id := transcriptionID(path)
	t.say(task.Task{ID: id, Doing: "Transcribing a recording", About: path}, asked)

	res, err := t.transcribe(ctx, v, id, path, asked)
	switch {
	case errors.Is(err, context.Canceled):
		// A transcription somebody stopped is one that is over, and the
		// recording is where the next round finds it.
		t.done(id)
	case err != nil:
		// The recording is left where the next round finds it. A store that
		// would not write and an index that would not answer are the machine,
		// and the recording has said nothing about itself.
		t.say(task.Task{ID: id, Doing: "Transcribing a recording", About: path, Failed: err.Error()}, asked)
	case res.Busy:
		t.done(id)
	default:
		t.done(id)
		t.recordAnswer(v, path)
		if !res.Silent && !res.Unopened {
			t.proofread(ctx, v, path, asked)
		}
	}
	return err
}

// ProofreaderReady says whether this installation has anything to put a
// transcript right with. A person is offered the run where it has.
func (t *TranscriptionWorker) ProofreaderReady() bool { return t.with.Proofreading.Named }

// Proofread puts one transcript right for a person who asked for it and says
// what came of asking: the run began, or the transcript needed nothing of it.
//
// The asking waits only for what a run settles before it puts a question to the
// proofreader, which is a few reads off the disk. The work itself carries on
// behind the answer, under the line in the list of what is being done that a
// proofreading of that recording carries.
func (t *TranscriptionWorker) Proofread(
	ctx context.Context,
	v domain.Vault,
	path string,
) (ProofreadTranscriptResult, error) {
	// A run says two things at most: that it began, and how it ended. The room
	// for both is here, so a run whose caller has gone says them and ends.
	said := make(chan outcome, 2)

	t.going.Add(1)
	go func() {
		defer t.going.Done()
		res, err := t.proofreadTranscript(t.context(), v, path, true, func(began ProofreadTranscriptResult) {
			said <- outcome{res: began}
		})
		said <- outcome{res: res, err: err}
	}()

	select {
	case one := <-said:
		return one.res, one.err
	case <-ctx.Done():
		return ProofreadTranscriptResult{Path: path}, ctx.Err()
	}
}

// outcome is what a run putting a transcript right says about itself: what it
// found, and what stopped it.
type outcome struct {
	res ProofreadTranscriptResult
	err error
}

// proofread puts a transcript right, where an installation asked for its
// transcripts to be put right on their own.
func (t *TranscriptionWorker) proofread(ctx context.Context, v domain.Vault, path string, asked bool) {
	if !t.with.Proofreading.Automatically {
		return
	}
	_, _ = t.proofreadTranscript(ctx, v, path, asked, nil)
}

// proofreadTranscript puts one transcript right with the profile named for
// speech.
//
// A transcript whose proofreading failed is the transcript as it was
// transcribed, and the failure stands in the list of what is being done until
// somebody reads it. Work a person asked for is in that list from the moment
// they asked.
func (t *TranscriptionWorker) proofreadTranscript(
	ctx context.Context,
	v domain.Vault,
	path string,
	asked bool,
	began func(ProofreadTranscriptResult),
) (ProofreadTranscriptResult, error) {
	said := t.with.Proofreading

	id := proofreadingID(path)
	fail := func(err error) {
		t.say(task.Task{
			ID: id, Doing: "Proofreading a transcript", About: path, Failed: err.Error(),
		}, asked)
	}
	if asked {
		t.say(task.Task{ID: id, Doing: "Proofreading a transcript", About: path}, asked)
	}

	by, err := said.By(proofread.SpeechInstruction)
	if err != nil {
		fail(err)
		return ProofreadTranscriptResult{Path: path}, err
	}
	if by == nil {
		t.done(id)
		return ProofreadTranscriptResult{Path: path}, nil
	}

	var once sync.Once
	right := NewProofreadTranscript(t.with.Readers, t.with.Derived, by)
	right.BatchSize = said.Batch
	right.Overlap = said.Overlap
	right.InFlight = said.InFlight
	right.Cut = t.Cut
	right.OnProgress = func(res ProofreadTranscriptResult) {
		// Progress is reported once there is a question to put, so the first of
		// it is this run beginning.
		once.Do(func() {
			if began != nil {
				began(res)
			}
		})
		t.say(task.Task{
			ID:    id,
			Doing: "Proofreading a transcript",
			About: path,
			Count: int64(res.Read),
			Total: int64(res.Lines),
		}, asked)
	}
	res, err := right.Execute(ctx, v, path)

	switch {
	case err != nil && !errors.Is(err, context.Canceled):
		fail(err)
	case res.Busy && !asked:
		// The transcript is held by another run, and that run is the one whose
		// progress the list carries.
	default:
		t.done(id)
	}
	return res, err
}

// TakingUp puts right the transcripts of these vaults that stand short of their
// last line, once, behind the caller.
//
// A proofreading stands at the line it reached, so a run that ended among the
// batches is taken up at that line. A transcript no proofreader has been over
// stands at its first line and is put right whole. Which transcripts a vault
// holds is a question the index already answers.
func (t *TranscriptionWorker) TakingUp(
	ctx context.Context,
	known port.SourceQueries,
	vaults ...domain.Vault,
) {
	if !t.with.Proofreading.Automatically || known == nil {
		return
	}
	t.going.Add(1)
	go func() {
		defer t.going.Done()
		for _, v := range vaults {
			recognised, err := known.Recognised(ctx, v.ID, domain.KindRecording)
			if err != nil {
				continue
			}
			for _, said := range recognised {
				if ctx.Err() != nil {
					return
				}
				t.proofread(ctx, v, said.Path, false)
			}
		}
	}()
}

// transcribe is the work itself: this machine's models held, what is missing
// arriving, and then the recording transcribed.
func (t *TranscriptionWorker) transcribe(
	ctx context.Context,
	v domain.Vault,
	id, path string,
	asked bool,
) (res TranscribeResult, err error) {
	// One run holds the models on a machine: a scan being recognised holds them,
	// and this waits for it.
	release, err := models.acquire(ctx, asked, func() {
		t.say(task.Task{ID: id, Doing: "Waiting for the models", About: path}, asked)
	})
	if err != nil {
		return res, err
	}
	defer release()

	// Getting the models is a step of its own and stands under its own name.
	// Which file is coming down, and how much of it, is known once one is.
	t.say(task.Task{ID: id, Doing: "Fetching models"}, asked)
	by, letGo, err := t.with.Runtime.Open(ctx, func(what string, done, total int64) {
		// The count is bytes and says so, and the sizes a person reads them in
		// are the window's to write.
		t.say(task.Task{
			ID: id, Doing: "Fetching models", About: what,
			Count: done, Total: total, Unit: task.Bytes,
		}, asked)
	})
	if err != nil {
		return res, fmt.Errorf("%w: %w", errNothingTranscribes, err)
	}
	// Letting the models go is what leaves the machine able to transcribe again.
	// A machine that will not is worth saying, and the transcript stands.
	defer func() {
		if why := letGo(); why != nil {
			err = errors.Join(err, fmt.Errorf("letting the models go: %w", why))
		}
	}()

	t.say(task.Task{ID: id, Doing: "Transcribing a recording", About: path}, asked)
	listen := NewTranscribe(t.with.Readers, t.with.Sources, t.with.Derived, by)
	listen.Cut = t.Cut
	listen.OnProgress = func(res TranscribeResult) {
		t.say(task.Task{
			ID:    id,
			Doing: "Transcribing a recording",
			About: path,
			Count: int64(res.Heard / 1000),
			Total: int64(res.Length / 1000),
			Unit:  task.Seconds,
		}, asked)
	}
	return listen.Execute(ctx, v, path)
}

// recordAnswer puts a recording out of the queue's reach for the life of this
// run. The answer itself is a file in the vault, so the next run reads it back
// and hands the recording over once and no more.
func (t *TranscriptionWorker) recordAnswer(v domain.Vault, path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.answered[named(v, path)] = true
}

// Forget puts a recording back within the queue's reach. The answer it gave is
// gone from the vault, and the queue hands it over again.
func (t *TranscriptionWorker) Forget(v domain.Vault, path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.answered, named(v, path))
}

func (t *TranscriptionWorker) context() context.Context {
	if t.under == nil {
		return context.Background()
	}
	return t.under
}

// say puts this transcription in the list of what is being done. Work nobody
// asked for is shown once it has lasted, and work a person started is shown at
// once.
func (t *TranscriptionWorker) say(at task.Task, asked bool) {
	at.Asked = asked
	if t.with.Tasks != nil {
		t.with.Tasks.Set(at)
	}
}

func (t *TranscriptionWorker) done(id string) {
	if t.with.Tasks != nil {
		t.with.Tasks.Done(id)
	}
}
