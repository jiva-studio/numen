package transcribe

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/transcript"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/ports"
)

func (t *Transcriber) transcribe(ctx context.Context, vault domain.VaultID, path string, read readRecording, release func()) error {
	err := t.run(ctx, vault, path, read)
	release()
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
	isWhole, err := j.store.hasWhole(ctx, j.key)
	if err != nil {
		return err
	}
	if isWhole {
		return t.putWholeVersion(ctx, j)
	}

	if isSkipped, err := t.settleFailure(ctx, j); err != nil || isSkipped {
		return err
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
	if err := t.hear(ctx, j, model, raw); err != nil {
		return rememberFailure(ctx, j, err)
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

func (t *Transcriber) settleFailure(ctx context.Context, j *job) (bool, error) {
	if t.isBackground(getTaskID(j.vault, j.path)) {
		return j.store.hasFailed(ctx, j.key)
	}
	return false, j.store.clearFailure(ctx, j.key)
}

func rememberFailure(ctx context.Context, j *job, err error) error {
	if errors.Is(err, context.Canceled) {
		return err
	}
	return errors.Join(err, j.store.writeFailure(ctx, j.key, err))
}

func (t *Transcriber) hear(ctx context.Context, j *job, model ports.Recognizer, raw []byte) (err error) {
	recording, err := model.Open(ctx, raw)
	if err != nil {
		return fmt.Errorf("%s cannot be opened: %w", j.path, err)
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="closing recording in defer releases audio resources"
	defer func() { err = errors.Join(err, recording.Close()) }()
	return t.hearSegments(ctx, j, model, recording)
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
	turn, err := host.InferenceQueue(t.queue.EnqueueWork).Acquire(ctx, host.Work{IsUserStarted: !t.isBackground(getTaskID(j.vault, j.path)), NotifyWaiting: func() {
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

func (t *Transcriber) hearSegments(ctx context.Context, j *job, model ports.Recognizer, recording ports.Recording) error {
	from, err := j.store.readReachedPosition(ctx, j.key)
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
	ctx context.Context, model ports.Recognizer, recording ports.Recording, path string, from int,
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
	task.IsUserStarted = !t.isBackground(task.ID)
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="best-effort stderr diagnostic log does not affect transcription control flow"
	if err := t.queue.ReportTask(context.WithoutCancel(ctx), task); err != nil {
		t.reportError("show the task "+task.ID, err)
	}
}

func (t *Transcriber) finishTask(ctx context.Context, vault domain.VaultID, path string) {
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="best-effort stderr diagnostic log does not affect transcription control flow"
	if err := t.queue.FinishTask(context.WithoutCancel(ctx), getTaskID(vault, path)); err != nil {
		t.reportError("finish the task of "+path, err)
	}
}
