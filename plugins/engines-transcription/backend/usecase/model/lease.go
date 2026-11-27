package model

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/speech"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/ports"
)

// lease is the open model as the transcription use case holds it. It counts the recordings open on
// the model and the transcriptions running on it, so a model the holder retired closes once the last
// of them ends, and a recording requested after the retirement opens on the current model.
type lease struct {
	holder    *Holder
	model     Model
	mu        sync.Mutex
	uses      int
	isRetired bool
	isClosed  bool
}

func (l *lease) Open(ctx context.Context, raw []byte) (ports.Recording, error) {
	if !l.acquire(false) {
		return l.openOnCurrent(ctx, raw)
	}
	recording, err := l.model.Open(ctx, raw)
	if err != nil {
		l.release()
		return nil, err
	}
	return &leasedRecording{Recording: recording, lease: l, once: sync.Once{}}, nil
}

func (l *lease) Transcribe(ctx context.Context, audio speech.Audio) (string, error) {
	if !l.acquire(true) {
		current, err := l.holder.GetRecognizer(ctx)
		if err != nil {
			return "", err
		}
		return current.Transcribe(ctx, audio)
	}
	defer l.release()
	return l.model.Transcribe(ctx, audio)
}

func (l *lease) openOnCurrent(ctx context.Context, raw []byte) (ports.Recording, error) {
	current, err := l.holder.GetRecognizer(ctx)
	if err != nil {
		return nil, err
	}
	return current.Open(ctx, raw)
}

// acquire counts one more use of the model. A retired model takes no new recording but still
// takes the transcriptions of the recordings open on it until it closes.
func (l *lease) acquire(isRetiredAllowed bool) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.isClosed || (l.isRetired && !isRetiredAllowed) {
		return false
	}
	l.uses++
	return true
}

func (l *lease) release() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.uses--
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="Stderr logging of close failure does not affect model state."
	if err := l.closeUnusedLocked(); err != nil {
		l.holder.reportCloseError(err)
	}
}

func (l *lease) retire() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.isRetired = true
	return l.closeUnusedLocked()
}

func (l *lease) closeUnusedLocked() error {
	if !l.isRetired || l.isClosed || l.uses > 0 {
		return nil
	}
	l.isClosed = true
	return l.model.Close()
}

// leasedRecording is a recording that ends its use of the model when it closes.
type leasedRecording struct {
	ports.Recording
	lease *lease
	once  sync.Once
}

func (r *leasedRecording) Close() error {
	err := r.Recording.Close()
	r.once.Do(r.lease.release)
	return err
}
