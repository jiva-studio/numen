package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcript"
)

const (
	taskTranscribing = "Transcribing a recording"
	taskWaiting      = "Waiting for the models"
	taskLoading      = "Loading the models"
)

// batchSize is how many stretches of speech are heard before what was heard is written down and
// put as the version so far.
const batchSize = 16

// Files reads the files of the vaults the host knows.
type Files interface {
	ReadFile(ctx context.Context, vault domain.VaultID, path string) ([]byte, error)
}

// Engines opens the model a recording is heard with. It is opened once and kept.
type Engines func(ctx context.Context) (engine.TranscriberPort, error)

// Transcriber hears the recordings a person asks for, one at a time through the core's queue, and
// reports each as a task and as a timed text version that grows as the words are written down.
type Transcriber struct {
	files   Files
	stores  host.Store
	texts   host.Texts
	queue   host.Queue
	engines Engines

	// ctx is what every transcription runs under, and it ends when the plugin stops.
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.Mutex
	running map[string]bool
}

// NewTranscriber is the transcriber over what the host gives the plugin.
func NewTranscriber(files Files, stores host.Store, texts host.Texts, queue host.Queue, engines Engines) *Transcriber {
	ctx, cancel := context.WithCancel(context.Background())
	return &Transcriber{
		files: files, stores: stores, texts: texts, queue: queue, engines: engines,
		ctx: ctx, cancel: cancel, running: map[string]bool{},
	}
}

// Close stops every transcription and waits for them.
func (t *Transcriber) Close() {
	t.cancel()
	t.wg.Wait()
}

// Wait waits for every transcription started.
func (t *Transcriber) Wait() { t.wg.Wait() }

// getTaskID is what the transcription of one recording is called in the list of tasks.
func getTaskID(path string) string { return "transcription:" + path }

// Start begins transcribing a recording and returns once the task is shown. A recording already
// being transcribed is left to the run that has it.
func (t *Transcriber) Start(ctx context.Context, vault domain.VaultID, path string) error {
	key, isClaimed := t.claim(vault, path)
	if !isClaimed {
		return nil
	}
	if err := t.queue.ReportTask(ctx, host.Task{ID: getTaskID(path), Title: taskTranscribing, About: path, IsUserStarted: true}); err != nil {
		t.clearRunning(key)
		return err
	}
	t.wg.Go(func() {
		defer t.clearRunning(key)
		_ = t.transcribe(t.ctx, vault, path)
	})
	return nil
}

// errAlreadyTranscribing is a recording another run is transcribing.
var errAlreadyTranscribing = errors.New("the recording is already being transcribed")

// Transcribe hears a recording and returns once its transcript is written down, with the error the
// transcription stopped on. It ends when ctx ends or the plugin stops.
func (t *Transcriber) Transcribe(ctx context.Context, vault domain.VaultID, path string) error {
	key, isClaimed := t.claim(vault, path)
	if !isClaimed {
		return fmt.Errorf("%s: %w", path, errAlreadyTranscribing)
	}
	defer t.clearRunning(key)
	t.wg.Add(1)
	defer t.wg.Done()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer context.AfterFunc(t.ctx, cancel)() //nolint:contextcheck // the plugin stopping ends the command too

	if err := t.queue.ReportTask(ctx, host.Task{ID: getTaskID(path), Title: taskTranscribing, About: path, IsUserStarted: true}); err != nil {
		return err
	}
	return t.transcribe(ctx, vault, path)
}

// claim marks the recording as being transcribed, and is false where a run already has it.
func (t *Transcriber) claim(vault domain.VaultID, path string) (string, bool) {
	key := string(vault) + "\x00" + path
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running[key] {
		return key, false
	}
	t.running[key] = true
	return key, true
}

func (t *Transcriber) clearRunning(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.running, key)
}

// transcribe is one recording heard and reported. A failure stays in the list of tasks under the
// recording until the next run of it.
func (t *Transcriber) transcribe(ctx context.Context, vault domain.VaultID, path string) error {
	err := t.run(ctx, vault, path)
	switch {
	case err == nil, errors.Is(err, context.Canceled):
		t.finishTask(ctx, path)
	default:
		t.reportTask(ctx, host.Task{ID: getTaskID(path), Title: taskTranscribing, About: path, Error: err.Error(), IsUserStarted: true})
	}
	return err
}

// job is the state of one recording being heard.
type job struct {
	vault    domain.VaultID
	path     string
	key      string
	fileHash string
	store    transcripts
	texts    host.Texts
	turn     host.Turn
}

func (t *Transcriber) run(ctx context.Context, vault domain.VaultID, path string) error {
	raw, err := t.files.ReadFile(ctx, vault, path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	fileHash := domain.HashContent(raw)
	stores, texts := t.stores, t.texts
	if vault != "" {
		stores, texts = stores.With(vault), texts.With(vault)
	}
	j := &job{
		vault: vault, path: path, key: getTranscriptionKey(fileHash), fileHash: fileHash,
		store: transcripts{store: stores}, texts: texts,
	}
	if isWhole, err := j.store.hasWhole(ctx, j.key); err != nil || isWhole {
		if err != nil {
			return err
		}
		return t.putWholeVersion(ctx, j)
	}

	if err := t.takeTurn(ctx, j); err != nil {
		return err
	}
	defer func() { j.turn.Release() }()

	t.reportTask(ctx, host.Task{ID: getTaskID(path), Title: taskLoading, About: path, IsUserStarted: true})
	model, err := t.engines(ctx)
	if err != nil {
		return fmt.Errorf("open the model: %w", err)
	}
	recording, err := model.Open(ctx, raw)
	if err != nil {
		return fmt.Errorf("%s cannot be opened: %w", path, err)
	}
	defer func() { _ = recording.Close() }()

	if err := t.hearSegments(ctx, j, model, recording); err != nil {
		return err
	}
	err = j.store.finish(ctx, j.key)
	if errors.Is(err, host.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return t.putVersion(ctx, j, getFileName(j.key))
}

// putWholeVersion puts the version of a transcript a run before this one finished, where the host
// has no record of it.
func (t *Transcriber) putWholeVersion(ctx context.Context, j *job) error {
	listing, err := j.texts.ListVersions(ctx, j.path)
	if err != nil {
		return err
	}
	for _, version := range listing.Versions {
		if version.Key == j.key && version.PluginID == PluginID {
			return nil
		}
	}
	return t.putVersion(ctx, j, getFileName(j.key))
}

// putVersion records the transcript in the file of the plugin's folder named.
func (t *Transcriber) putVersion(ctx context.Context, j *job, file string) error {
	_, err := j.texts.PutVersion(ctx, host.TextPut{
		Path: j.path, Key: j.key, Kind: textKind, SourceHash: j.fileHash, File: file,
	})
	return err
}

// takeTurn waits for the recording's turn in the queue.
func (t *Transcriber) takeTurn(ctx context.Context, j *job) error {
	turn, err := t.queue.EnqueueWork(ctx, host.Work{IsUserStarted: true, OnWaiting: func() {
		t.reportTask(ctx, host.Task{ID: getTaskID(j.path), Title: taskWaiting, About: j.path, IsUserStarted: true})
	}})
	if err != nil {
		return err
	}
	j.turn = turn
	return nil
}

// hearSegments hears the recording from where a run before this one stopped until it reaches no
// further, writing and putting each batch as it is heard. Work waiting for the queue takes its turn
// between two batches.
func (t *Transcriber) hearSegments(ctx context.Context, j *job, model engine.TranscriberPort, recording engine.Recording) error {
	from, err := j.store.readReached(ctx, j.key)
	if err != nil {
		return err
	}
	length := recording.GetDuration()
	for {
		t.reportProgress(ctx, j.path, from, length)
		cues, reached, err := hearBatch(ctx, model, recording, j.path, from)
		if err != nil {
			return err
		}
		if reached <= from {
			return nil
		}
		if err := j.store.appendBatch(ctx, j.key, cues, reached); err != nil {
			return err
		}
		from = reached
		if err := t.putVersion(ctx, j, getPartialName(j.key)); err != nil {
			return err
		}
		if err := t.yieldTurn(ctx, j); err != nil {
			return err
		}
	}
}

// yieldTurn lets work waiting for the queue run, and waits for the recording's turn again.
func (t *Transcriber) yieldTurn(ctx context.Context, j *job) error {
	select {
	case <-j.turn.WatchWaiting():
	default:
		return nil
	}
	j.turn.Release()
	return t.takeTurn(ctx, j)
}

// hearBatch hears the next stretches of speech after from, and returns their cues with the furthest
// moment they reach. A recording with nothing after from returns no cues and from itself.
func hearBatch(
	ctx context.Context, model engine.TranscriberPort, recording engine.Recording, path string, from int,
) ([]transcript.Cue, int, error) {
	segments, err := recording.GetSegments(ctx, from, batchSize)
	if err != nil {
		return nil, from, fmt.Errorf("transcribe %s at %s: %w", path, transcript.FormatTimestamp(from), err)
	}
	reached := from
	cues := make([]transcript.Cue, 0, len(segments))
	for _, audio := range segments {
		words, err := model.Transcribe(ctx, audio)
		if err != nil {
			return nil, from, fmt.Errorf("transcribe %s at %s: %w", path, transcript.FormatTimestamp(audio.From), err)
		}
		cues = append(cues, transcript.Cue{Text: words, From: audio.From, To: audio.To})
		reached = max(reached, audio.To)
	}
	return cues, reached, nil
}

func (t *Transcriber) reportProgress(ctx context.Context, path string, heard, length int) {
	t.reportTask(ctx, host.Task{
		ID: getTaskID(path), Title: taskTranscribing, About: path,
		Done: int64(heard / 1000), Total: int64(length / 1000), Unit: host.UnitSeconds, IsUserStarted: true,
	})
}

// reportTask shows the task. A task the host could not be told of is shown by the next report.
func (t *Transcriber) reportTask(ctx context.Context, task host.Task) {
	_ = t.queue.ReportTask(context.WithoutCancel(ctx), task)
}

// finishTask takes the recording's task out of the list. One the host could not be told to take out
// goes when the plugin stops.
func (t *Transcriber) finishTask(ctx context.Context, path string) {
	_ = t.queue.FinishTask(context.WithoutCancel(ctx), getTaskID(path))
}
