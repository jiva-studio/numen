package container

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/transcription"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// TranscriberReady says whether a recording could be listened to now without
// waiting for anything to arrive. Which models those are is settled here, with
// every other choice of adapter.
func (c Config) TranscriberReady() bool { return transcription.Ready(c.Transcription) }

// Transcriber is what listens to a recording on this machine, opened now: the
// transcriber, what gives it back, and why there is none.
//
// It waits for whatever is missing, so it is for a terminal, where waiting is
// what a person came for. A window asks Transcribing instead.
func (c Config) Transcriber(ctx context.Context) (transcriber port.Transcriber, close func() error, why error) {
	models, err := transcription.Open(ctx, c.Transcription)
	if err != nil {
		return nil, nil, err
	}
	return models, models.Close, nil
}

// errNothingListens is a run that reached nothing: what transcribes a recording
// is not on this machine. It is about the machine and not about the file, so a
// queue leaves the work where it is and comes back to it.
var errNothingListens = errors.New("nothing to transcribe with")

// hearing is what one recording's transcription is called, wherever it is
// shown. One recording is one line, and it replaces itself as the words are
// written down.
func hearing(path string) string { return "hearing-" + path }

// heavy is the one heavy run this machine does at a time. Reading a scan and
// listening to a recording each hold the models and the processor, and they
// take turns.
var heavy turn

// The queues a run waits in, the first of them served first. Work a person is
// sitting in front of goes before work the vault set itself.
const (
	queueAsked = iota
	queueUnasked
	queues
)

// A turn is handed out to one run at a time. A run waits in the queue its work
// belongs to and is served in the order it arrived there.
type turn struct {
	mu      sync.Mutex
	held    bool
	waiting [queues][]chan struct{}
}

// take waits for this machine's turn at the models and hands back what gives
// the turn up. waiting is called where the turn is not free, so that a person
// watching is told why nothing is moving. A context that ends while waiting
// takes no turn.
func (g *turn) take(ctx context.Context, asked bool, waiting func()) (func(), error) {
	at := queueUnasked
	if asked {
		at = queueAsked
	}

	g.mu.Lock()
	if !g.held {
		g.held = true
		g.mu.Unlock()
		return g.give, nil
	}
	stand := make(chan struct{})
	g.waiting[at] = append(g.waiting[at], stand)
	g.mu.Unlock()

	if waiting != nil {
		waiting()
	}
	select {
	case <-stand:
		return g.give, nil
	case <-ctx.Done():
		g.leave(at, stand)
		return nil, ctx.Err()
	}
}

// give hands the turn to whoever has waited longest in the first queue anybody
// stands in, and lets it go where nobody does.
func (g *turn) give() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for at := range g.waiting {
		if len(g.waiting[at]) > 0 {
			next := g.waiting[at][0]
			g.waiting[at] = g.waiting[at][1:]
			close(next)
			return
		}
	}
	g.held = false
}

// leave takes a run out of its queue. One already handed the turn holds it, and
// gives it on.
func (g *turn) leave(at int, stand chan struct{}) {
	g.mu.Lock()
	for i, one := range g.waiting[at] {
		if one == stand {
			g.waiting[at] = slices.Delete(g.waiting[at], i, i+1)
			g.mu.Unlock()
			return
		}
	}
	g.mu.Unlock()
	<-stand
	g.give()
}

// Transcribing listens to recordings behind whoever asked, and behind nobody.
//
// A recording carries no text of its own, so a vault of recordings is work to
// be done whether or not anybody asks. The hand asks through Start and the
// queue asks through Queue, and both arrive here.
type Transcribing struct {
	cfg     Config
	sources port.SourceRepository
	tasks   *task.Tasks

	// under is what every transcription runs under, and going is every
	// transcription that has not ended. One outlives the question that asked
	// for it and ends with the application, which waits here for what it is
	// still writing.
	under context.Context
	going sync.WaitGroup

	// open is what hears a recording, opened when there is one to hear, and
	// ready says whether opening it would wait for anything to arrive.
	open  listening
	ready func() bool

	// Cut makes a source's chunks from what has been heard of it. It is called
	// as speech is written down, so the beginning of a recording is searchable
	// while the end of it is still being heard.
	Cut func(ctx context.Context, v domain.Vault, path string) error

	mu      sync.Mutex
	running bool
	// asked is the recordings a person named that have not been heard yet.
	asked asked
	// answered is every recording this run has had an answer about, by vault
	// and path. Words, silence and a file that will not open are all answers,
	// and a recording that has answered is not offered again.
	answered map[string]bool
}

// listening is what hears a recording, opened when there is one to hear. It is
// told how far the fetching of what it needs has got.
type listening func(ctx context.Context, tell func(what string, done, total int64)) (port.Transcriber, func() error, error)

// Transcribing is the transcriber this installation offers, reporting itself
// into the list of what is being done.
func (c Config) Transcribing(ctx context.Context, sources port.SourceRepository, tasks *task.Tasks) *Transcribing {
	return &Transcribing{
		cfg: c, sources: sources, tasks: tasks, under: ctx,
		open: func(ctx context.Context, tell func(what string, done, total int64)) (port.Transcriber, func() error, error) {
			cfg := c.Transcription
			cfg.Fetching = tell
			models, err := transcription.Open(ctx, cfg)
			if err != nil {
				return nil, nil, err
			}
			return models, models.Close, nil
		},
		ready:    c.TranscriberReady,
		answered: map[string]bool{},
	}
}

// Ready says whether a recording could be listened to now without waiting for
// anything to arrive.
func (t *Transcribing) Ready() bool { return t.ready() }

// Wait is every transcription this started, ended. What they write goes into an
// index the application still holds open.
func (t *Transcribing) Wait() { t.going.Wait() }

// Start listens to one recording a person named, and says whether it began now
// or waits its turn.
//
// One at a time: the models hold a worker each. A recording named while one is
// being heard goes to the back of the line and is heard as soon as the turn is
// free, ahead of everything the vault set itself.
//
// It runs under the application, so whoever asked is answered at once and goes
// away while the listening carries on.
func (t *Transcribing) Start(v domain.Vault, path string) port.Taking {
	t.mu.Lock()
	t.asked.want(v, path)
	if t.running {
		t.mu.Unlock()
		return port.Queued
	}
	t.running = true
	t.mu.Unlock()

	t.going.Add(1)
	go func() {
		defer t.going.Done()
		defer t.release()
		t.drain(t.context())
	}()
	return port.Began
}

// Waiting is how many recordings a person named are still in line.
func (t *Transcribing) Waiting() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.asked.waiting()
}

// drain hears every recording a person named, in the order they named them. A
// recording is out of the line before it is heard, so one whose listening ends
// where nothing expected it to holds nothing afterwards.
func (t *Transcribing) drain(ctx context.Context) {
	for {
		t.mu.Lock()
		one, waiting := t.asked.take()
		t.mu.Unlock()
		if !waiting || ctx.Err() != nil {
			return
		}
		t.hear(ctx, one.vault, one.path, true)
	}
}

// Queue listens to every recording this vault holds no transcript for, one
// after another, until the context ends.
//
// The queue is not stored: which recordings owe their text is a question the
// index already answers, so a round interrupted by the application closing is
// taken up when it opens. It is asked again every so often, because a recording
// dropped into a folder is one the last round did not see.
func (t *Transcribing) Queue(
	ctx context.Context,
	known port.SourceQueries,
	every time.Duration,
	v domain.Vault,
) {
	t.going.Add(1)
	defer t.going.Done()
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
func (t *Transcribing) round(ctx context.Context, known port.SourceQueries, v domain.Vault) {
	if !t.claim() {
		// The hand is listening to something. What this round did not hand over
		// is handed over by the next one.
		return
	}
	defer t.release()

	t.drain(ctx)
	for _, path := range t.owing(ctx, known, v) {
		if ctx.Err() != nil {
			return
		}
		err := t.hear(ctx, v, path, false)
		if errors.Is(err, errNothingListens) {
			return
		}
		// A recording named while this round ran is heard before the next one
		// the vault owes.
		t.drain(ctx)
	}
}

// owing is the recordings this vault holds that no model has written the words
// of, and that this run has had no answer about.
//
// The index knows both: every recording it holds, and which of them stand on a
// text a producer made. What is left of the first by the second is the work.
func (t *Transcribing) owing(ctx context.Context, known port.SourceQueries, v domain.Vault) []string {
	if known == nil {
		return nil
	}
	held, err := known.Fingerprints(ctx, v.ID, domain.KindRecording)
	if err != nil {
		return nil
	}
	heard, err := known.Recognised(ctx, v.ID, domain.KindRecording)
	if err != nil {
		return nil
	}
	for _, said := range heard {
		delete(held, said.Path)
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	under := t.cfg.TranscribesUnder
	out := make([]string, 0, len(held))
	for path, ref := range held {
		if t.answered[named(v, path)] {
			continue
		}
		if under > 0 && ref.Size > under {
			// A recording this large is somebody's music or somebody's archive.
			// It is listened to when it is asked for by name.
			continue
		}
		out = append(out, path)
	}
	slices.Sort(out)
	return out
}

// claim takes this installation's one transcription, and says whether it was
// free.
func (t *Transcribing) claim() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running {
		return false
	}
	t.running = true
	return true
}

func (t *Transcribing) release() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = false
}

// hear is one recording, listened to and written down.
//
// asked says whether a person is waiting to be told it began. Whatever it ends
// as is recorded, and a recording that ended in an answer is not offered again:
// words, silence and a file nothing here can open are all answers, and only
// bytes somebody else holds mean come back later.
func (t *Transcribing) hear(ctx context.Context, v domain.Vault, path string, asked bool) error {
	id := hearing(path)
	t.say(task.Task{ID: id, Doing: "Transcribing a recording", About: path}, asked)

	res, err := t.listen(ctx, v, id, path, asked)
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
			t.correct(ctx, v, path, asked)
		}
	}
	return err
}

// correct puts a transcript right, where a person configured something to
// proofread it with. An installation that named no profile, or asked for a
// transcript to be put right by hand, does nothing here.
//
// A transcript whose proofreading failed is the transcript as it was heard.
func (t *Transcribing) correct(ctx context.Context, v domain.Vault, path string, asked bool) {
	said := t.cfg.SpeechProofreading
	if !said.Automatically {
		return
	}

	id := correcting(path)
	fail := func(err error) {
		t.say(task.Task{
			ID: id, Doing: "Proofreading a transcript", About: path, Failed: err.Error(),
		}, asked)
	}

	by, err := t.cfg.Proofreader(said.With, proofread.SpeechInstruction)
	if err != nil {
		fail(err)
		return
	}
	if by == nil {
		return
	}

	profile := t.cfg.Proofreading.Profiles[said.With]

	res, err := source.PutRight{
		Readers:   t.cfg.VaultReaders(),
		Derived:   t.cfg.DerivedStores(),
		By:        by,
		BatchSize: profile.BatchSize,
		Overlap:   profile.Overlap,
		InFlight:  profile.InFlight,
		Cut:       t.Cut,
		OnProgress: func(res source.PutRightResult) {
			// A transcript already put right to its last line is one nobody is
			// waiting on, and it is shown nowhere.
			if res.Read >= res.Lines {
				return
			}
			t.say(task.Task{
				ID:    id,
				Doing: "Proofreading a transcript",
				About: path,
				Done:  int64(res.Read),
				Total: int64(res.Lines),
			}, asked)
		},
	}.Execute(ctx, v, path)

	switch {
	case res.Busy:
		// The transcript is held by another run, and that run is the one whose
		// progress the list carries.
	case err == nil, errors.Is(err, context.Canceled):
		t.done(id)
	default:
		fail(err)
	}
}

// TakingUp puts right what a run before this one stopped part way through.
//
// A proofreading stands at the line it reached, so the lines after it are asked
// about again once, when the application opens. Which transcripts those are is
// a question the store already answers, and no queue of them is kept.
func (t *Transcribing) TakingUp(
	ctx context.Context,
	known port.SourceQueries,
	vaults ...domain.Vault,
) {
	if !t.cfg.SpeechProofreading.Automatically || known == nil {
		return
	}
	t.going.Add(1)
	defer t.going.Done()
	for _, v := range vaults {
		heard, err := known.Recognised(ctx, v.ID, domain.KindRecording)
		if err != nil {
			continue
		}
		for _, said := range heard {
			if ctx.Err() != nil {
				return
			}
			t.correct(ctx, v, said.Path, false)
		}
	}
}

// listen is the work itself: this machine's turn at the models, what is missing
// arriving, and then the recording heard.
func (t *Transcribing) listen(
	ctx context.Context,
	v domain.Vault,
	id, path string,
	asked bool,
) (source.TranscribeResult, error) {
	var res source.TranscribeResult

	// One heavy run on a machine: a scan being read holds the turn, and this
	// waits for it.
	release, err := heavy.take(ctx, asked, func() {
		t.say(task.Task{ID: id, Doing: "Waiting for a turn at the models", About: path}, asked)
	})
	if err != nil {
		return res, err
	}
	defer release()

	models, close, err := t.open(ctx, func(what string, done, total int64) {
		// The count is bytes and says so, and the sizes a person reads them in
		// are the window's to write.
		t.say(task.Task{
			ID: id, Doing: "Fetching models", About: what,
			Done: done, Total: total, Counting: task.Bytes,
		}, asked)
	})
	if err != nil {
		return res, fmt.Errorf("%w: %w", errNothingListens, err)
	}
	defer close()

	t.say(task.Task{ID: id, Doing: "Transcribing a recording", About: path}, asked)
	return source.Transcribe{
		Readers: t.cfg.VaultReaders(),
		Sources: t.sources,
		Derived: t.cfg.DerivedStores(),
		By:      models,
		Cut:     t.Cut,
		OnProgress: func(res source.TranscribeResult) {
			t.say(task.Task{
				ID:       id,
				Doing:    "Transcribing a recording",
				About:    path,
				Done:     int64(res.Heard / 1000),
				Total:    int64(res.Length / 1000),
				Counting: task.Seconds,
			}, asked)
		},
	}.Execute(ctx, v, path)
}

// recordAnswer puts a recording out of the queue's reach for the life of this
// run. The answer itself is a file in the vault, so the next run reads it back
// and hands the recording over once and no more.
func (t *Transcribing) recordAnswer(v domain.Vault, path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.answered[named(v, path)] = true
}

// named is one recording of one vault, as the one string a set is keyed by.
func named(v domain.Vault, path string) string { return v.ID + "\x00" + path }

func (t *Transcribing) context() context.Context {
	if t.under == nil {
		return context.Background()
	}
	return t.under
}

// say puts this transcription in the list of what is being done. Work nobody
// asked for is shown once it has lasted, and work a person started is shown at
// once.
func (t *Transcribing) say(at task.Task, asked bool) {
	at.Asked = asked
	if t.tasks != nil {
		t.tasks.Set(at)
	}
}

func (t *Transcribing) done(id string) {
	if t.tasks != nil {
		t.tasks.Done(id)
	}
}
