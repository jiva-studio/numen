// Package transcribe transcribes the recordings a user requests, writes the transcripts into the
// plugin's own folder of the vault's store in the common format of text versions, and records each
// with the host as a version of kind "transcript".
package transcribe

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.engines.transcription"

const (
	taskTranscribing = "Transcribing a recording"
	taskWaiting      = "Waiting for the models"
	taskLoading      = "Loading the models"
)

// batchSize is how many speech segments are transcribed before the transcript is
// persisted and published as the current version.
const batchSize = 16

// Transcriber transcribes the recordings requested by the user, one at a time through the core's queue,
// and reports each as a task and as a timed text version that grows as the transcript is written.
type Transcriber struct {
	files   Files
	stores  func(domain.VaultID) Folder
	texts   func(domain.VaultID) host.TextVersions
	queue   host.Queue
	engines Engines
	report  func(what string, err error)

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
		files: files, stores: stores, texts: texts, queue: queue, engines: engines, report: nil,
		closing: make(chan struct{}), closeOnce: sync.Once{}, wg: sync.WaitGroup{},
		mu: sync.Mutex{}, isClosed: false, running: map[string]bool{},
	}
}

// SetReport sets the callback that receives background failures of the transcriber.
func (t *Transcriber) SetReport(report func(what string, err error)) {
	t.report = report
}

func (t *Transcriber) reportError(what string, err error) {
	if t.report != nil && err != nil {
		t.report(what, err)
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
	// ErrAlreadyTranscribing is returned for a recording another run is transcribing.
	ErrAlreadyTranscribing = errors.New("the recording is already being transcribed")
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
			t.reportError("transcribe "+path, err)
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
		return "", "", fmt.Errorf("%s: %w", path, ErrAlreadyTranscribing)
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
