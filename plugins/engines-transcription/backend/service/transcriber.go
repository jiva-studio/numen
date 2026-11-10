package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcript"
)

const (
	taskTranscribing = "Transcribing a recording"
	taskWaiting      = "Waiting for the models"
	taskLoading      = "Loading the models"
)

// batchSize is how many speech segments are transcribed before the transcript is
// persisted and published as the current version.
const batchSize = 16

// Files reads files from the vaults known to the host.
type Files interface {
	ReadFile(ctx context.Context, vault domain.VaultID, path string) ([]byte, error)
}

// activeVaultProvider is implemented by Files that can return the vault the host shows. A recording
// requested with an empty vault ID is then claimed under the id of that vault, the id a tool call
// names.
type activeVaultProvider interface {
	GetActiveVaultID(ctx context.Context) (domain.VaultID, error)
}

// Engines opens the model that transcribes a recording. The model is opened once and reused.
type Engines func(ctx context.Context) (engine.TranscriberPort, error)

// Transcriber transcribes the recordings requested by the user, one at a time through the core's queue,
// and reports each as a task and as a timed text version that grows as the transcript is written.
type Transcriber struct {
	files   Files
	stores  func(domain.VaultID) Folder
	texts   func(domain.VaultID) host.TextVersions
	queue   host.Queue
	engines Engines

	closing   chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup

	mu       sync.Mutex
	isClosed bool
	running  map[string]bool
}

// NewTranscriber is the transcriber over what the host gives the plugin.
func NewTranscriber(files Files, stores func(domain.VaultID) Folder, texts func(domain.VaultID) host.TextVersions, queue host.Queue, engines Engines) *Transcriber {
	return &Transcriber{
		files: files, stores: stores, texts: texts, queue: queue, engines: engines,
		closing: make(chan struct{}), closeOnce: sync.Once{}, wg: sync.WaitGroup{},
		mu: sync.Mutex{}, isClosed: false, running: map[string]bool{},
	}
}

// Close stops every transcription and waits for them. A transcription requested after Close is not
// started.
func (t *Transcriber) Close() {
	t.mu.Lock()
	t.isClosed = true
	t.mu.Unlock()
	// gomutants:disable-next-line STATEMENT_REMOVE reason="closing channel cancels in-flight transcriptions"
	t.closeOnce.Do(func() { close(t.closing) })
	// gomutants:disable-next-line STATEMENT_REMOVE reason="Wait synchronizes shutdown with active goroutines"
	t.wg.Wait()
}

func (t *Transcriber) withClosing(ctx context.Context) (context.Context, context.CancelFunc) {
	runCtx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-t.closing:
			// gomutants:disable-next-line STATEMENT_REMOVE reason="cancels runCtx when Close is called"
			cancel()
		case <-runCtx.Done():
		}
	}()
	return runCtx, cancel
}

func getTaskID(vault domain.VaultID, path string) string {
	return "transcription:" + string(vault) + ":" + path
}

var (
	// errAlreadyTranscribing is returned for a recording another run is transcribing.
	errAlreadyTranscribing = errors.New("the recording is already being transcribed")
	// errClosed is returned for a transcription requested after Close.
	errClosed = errors.New("the transcriber is closed")
)

// readRecording returns the bytes of the recording a run transcribes.
type readRecording func(ctx context.Context) ([]byte, error)

// Transcribe transcribes a recording and returns once its transcript is written, with the error the
// transcription stopped on. It ends when ctx ends or the plugin stops.
func (t *Transcriber) Transcribe(ctx context.Context, vault domain.VaultID, path string) error {
	claimed, key, err := t.claimOrFail(ctx, vault, path)
	if err != nil {
		return err
	}
	return t.runTranscription(ctx, key, claimed, path, func(ctx context.Context) ([]byte, error) {
		return t.files.ReadFile(ctx, claimed, path)
	})
}

// Start claims the recording and transcribes it in the background, and returns false when a run
// already has it. The recording is read in the background as well, and a failed read fails the
// task. Close cancels the transcription and waits for it.
func (t *Transcriber) Start(ctx context.Context, vault domain.VaultID, path string) (bool, error) {
	claimed, err := t.getClaimVault(ctx, vault)
	if err != nil {
		return false, err
	}
	key, isClaimed := t.claim(claimed, path)
	if !isClaimed {
		if t.isClosing() {
			return false, errClosed
		}
		return false, nil
	}
	if err := t.reportStart(ctx, claimed, path); err != nil {
		t.clearRunning(key)
		return false, err
	}
	go func() {
		// Run reports a failure as the task's error.
		// gomutants:disable-next-line BRANCH_IF,CONDITIONALS_NEGATION reason="best-effort stderr diagnostic log does not affect transcription control flow"
		if err := t.runTranscription(context.WithoutCancel(ctx), key, claimed, path, func(ctx context.Context) ([]byte, error) {
			return t.files.ReadFile(ctx, claimed, path)
		}); err != nil && !errors.Is(err, context.Canceled) {
			fmt.Fprintf(os.Stderr, "engines-transcription: transcribe %s: %v\n", path, err)
		}
	}()
	return true, nil
}

func (t *Transcriber) claimOrFail(ctx context.Context, vault domain.VaultID, path string) (domain.VaultID, string, error) {
	claimed, err := t.getClaimVault(ctx, vault)
	if err != nil {
		return "", "", err
	}
	key, isClaimed := t.claim(claimed, path)
	switch {
	case isClaimed:
		return claimed, key, nil
	case t.isClosing():
		return "", "", fmt.Errorf("%s: %w", path, errClosed)
	default:
		return "", "", fmt.Errorf("%s: %w", path, errAlreadyTranscribing)
	}
}

func (t *Transcriber) runTranscription(ctx context.Context, key string, vault domain.VaultID, path string, read readRecording) error {
	defer t.clearRunning(key)
	ctx, cancel := t.withClosing(ctx)
	defer cancel()

	if err := t.reportStart(ctx, vault, path); err != nil {
		return err
	}
	return t.transcribe(ctx, vault, path, read)
}

func (t *Transcriber) reportStart(ctx context.Context, vault domain.VaultID, path string) error {
	return t.queue.ReportTask(ctx, host.Task{
		ID: getTaskID(vault, path), Title: taskTranscribing, Detail: path, IsUserStarted: true,
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="indeterminate task progress uses zero counts"
		CompletedCount: 0, TotalCount: 0, Unit: host.UnitThings, FailureReason: "",
	})
}

func (t *Transcriber) claim(vault domain.VaultID, path string) (string, bool) {
	key := getRunKey(vault, path)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.isClosed || t.running[key] {
		return key, false
	}
	t.running[key] = true
	t.wg.Add(1)
	return key, true
}

func getRunKey(vault domain.VaultID, path string) string { return string(vault) + "\x00" + path }

func (t *Transcriber) isRunning(vault domain.VaultID, path string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running[getRunKey(vault, path)]
}

func (t *Transcriber) findLiveVersionOfWholeTranscript(ctx context.Context, vault domain.VaultID, path string) (domain.TextVersion, bool, error) {
	listing, err := t.texts(vault).ListVersions(ctx, path)
	if err != nil {
		return domain.TextVersion{}, false, err
	}
	byID := make(map[domain.TextVersionID]domain.TextVersion, len(listing.Versions))
	for _, version := range listing.Versions {
		byID[version.GetID()] = version
	}
	live, isLive := byID[listing.Live]
	seenIDs := map[domain.TextVersionID]bool{}
	for version, isFound := live, isLive; isFound && !seenIDs[version.GetID()]; version, isFound = byID[version.Source] {
		if version.PluginID == PluginID && isWholeFile(version) {
			return live, true, nil
		}
		seenIDs[version.GetID()] = true
	}
	return domain.TextVersion{}, false, nil
}

func (t *Transcriber) getClaimVault(ctx context.Context, vault domain.VaultID) (domain.VaultID, error) {
	resolver, isResolver := t.files.(activeVaultProvider)
	if vault != "" || !isResolver {
		return vault, nil
	}
	shown, err := resolver.GetActiveVaultID(ctx)
	if err != nil {
		return "", fmt.Errorf("get the shown vault: %w", err)
	}
	return shown, nil
}

func (t *Transcriber) clearRunning(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.running, key)
	t.wg.Done()
}

func (t *Transcriber) isClosing() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.isClosed
}

func (t *Transcriber) transcribe(ctx context.Context, vault domain.VaultID, path string, read readRecording) error {
	err := t.run(ctx, vault, path, read)
	switch {
	case err == nil, errors.Is(err, context.Canceled):
		t.finishTask(ctx, vault, path)
	default:
		t.reportTask(ctx, host.Task{
			ID: getTaskID(vault, path), Title: taskTranscribing, Detail: path, FailureReason: err.Error(), IsUserStarted: true,
			// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="indeterminate task progress uses zero counts"
			CompletedCount: 0, TotalCount: 0, Unit: host.UnitThings,
		})
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
	texts    host.TextVersions
	turn     host.Turn
}

func (t *Transcriber) run(ctx context.Context, vault domain.VaultID, path string, read readRecording) (err error) {
	raw, err := read(ctx)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	fileHash := domain.HashContent(raw)
	stores, texts := t.stores(vault), t.texts(vault)
	j := &job{
		vault: vault, path: path, key: getTranscriptionKey(fileHash), fileHash: fileHash,
		store: transcripts{store: stores}, texts: texts, turn: nil,
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
	// gomutants:disable-next-line STATEMENT_REMOVE reason="releasing the queue turn in defer prevents deadlocking subsequent work"
	defer func() { err = errors.Join(err, j.turn.Release()) }()

	// gomutants:disable-next-line STATEMENT_REMOVE reason="progress notification before opening model"
	t.reportTask(ctx, host.Task{
		ID: getTaskID(vault, path), Title: taskLoading, Detail: path, IsUserStarted: true,
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="indeterminate task progress uses zero counts"
		CompletedCount: 0, TotalCount: 0, Unit: host.UnitThings, FailureReason: "",
	})
	model, err := t.engines(ctx)
	if err != nil {
		return fmt.Errorf("open the model: %w", err)
	}
	recording, err := model.Open(ctx, raw)
	if err != nil {
		return fmt.Errorf("%s cannot be opened: %w", path, err)
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="closing recording in defer releases audio resources"
	defer func() { err = errors.Join(err, recording.Close()) }()

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

func (t *Transcriber) putWholeVersion(ctx context.Context, j *job) error {
	listing, err := j.texts.ListVersions(ctx, j.path)
	if err != nil {
		return err
	}
	for _, version := range listing.Versions {
		if version.Key == j.key && version.PluginID == PluginID && isWholeFile(version) {
			return nil
		}
	}
	return t.putVersion(ctx, j, getFileName(j.key))
}

func (t *Transcriber) putVersion(ctx context.Context, j *job, file string) error {
	_, err := j.texts.PutVersion(ctx, host.TextPut{
		Path: j.path, Key: j.key, Source: "", Kind: textKind, InputHash: j.fileHash, File: file,
	})
	return err
}

func (t *Transcriber) takeTurn(ctx context.Context, j *job) error {
	turn, err := t.queue.EnqueueWork(ctx, host.Work{IsUserStarted: true, OnWaiting: func() {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="best-effort waiting task status notification"
		t.reportTask(ctx, host.Task{
			ID: getTaskID(j.vault, j.path), Title: taskWaiting, Detail: j.path, IsUserStarted: true,
			// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="indeterminate task progress uses zero counts"
			CompletedCount: 0, TotalCount: 0, Unit: host.UnitThings, FailureReason: "",
		})
	}})
	if err != nil {
		return err
	}
	j.turn = turn
	return nil
}

func (t *Transcriber) hearSegments(ctx context.Context, j *job, model engine.TranscriberPort, recording engine.Recording) error {
	from, err := j.store.readReached(ctx, j.key)
	if err != nil {
		return err
	}
	length := recording.GetDuration()
	for {
		t.reportProgress(ctx, j.vault, j.path, from, length)
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

func (t *Transcriber) yieldTurn(ctx context.Context, j *job) error {
	select {
	case <-j.turn.WatchWaiting():
	default:
		return nil
	}
	if err := j.turn.Release(); err != nil {
		return err
	}
	return t.takeTurn(ctx, j)
}

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

func (t *Transcriber) reportProgress(ctx context.Context, vault domain.VaultID, path string, heard, length int) {
	t.reportTask(ctx, host.Task{
		ID: getTaskID(vault, path), Title: taskTranscribing, Detail: path,
		CompletedCount: int64(heard / 1000), TotalCount: int64(length / 1000), Unit: host.UnitSeconds, IsUserStarted: true, FailureReason: "",
	})
}

func (t *Transcriber) reportTask(ctx context.Context, task host.Task) {
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="best-effort stderr diagnostic log does not affect transcription control flow"
	if err := t.queue.ReportTask(context.WithoutCancel(ctx), task); err != nil {
		fmt.Fprintf(os.Stderr, "engines-transcription: show the task %s: %v\n", task.ID, err)
	}
}

func (t *Transcriber) finishTask(ctx context.Context, vault domain.VaultID, path string) {
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="best-effort stderr diagnostic log does not affect transcription control flow"
	if err := t.queue.FinishTask(context.WithoutCancel(ctx), getTaskID(vault, path)); err != nil {
		fmt.Fprintf(os.Stderr, "engines-transcription: finish the task of %s: %v\n", path, err)
	}
}
