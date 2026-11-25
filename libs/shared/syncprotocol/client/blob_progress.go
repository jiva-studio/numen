package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

// ErrTransferStalled means a transfer of a blob moved no byte for the idle time of the client.
var ErrTransferStalled = errors.New("the transfer of a blob stalled")

// progressTimer cancels a transfer that moves no byte for its idle time.
type progressTimer struct {
	idle   time.Duration
	timer  *time.Timer
	cancel context.CancelCauseFunc
	mu     sync.Mutex
	isDone bool
}

func newProgressTimer(idle time.Duration, cancel context.CancelCauseFunc) *progressTimer {
	t := &progressTimer{idle: idle, cancel: cancel, mu: sync.Mutex{}, isDone: false, timer: nil}
	t.timer = time.AfterFunc(idle, func() { cancel(ErrTransferStalled) })
	return t
}

func (t *progressTimer) touch() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.isDone {
		t.timer.Reset(t.idle)
	}
}

func (t *progressTimer) stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.isDone = true
	t.timer.Stop()
}

func readStall(ctx context.Context, err error) error {
	if err != nil && errors.Is(context.Cause(ctx), ErrTransferStalled) {
		return ErrTransferStalled
	}
	return err
}

type progressRequestBody struct {
	source io.ReadCloser
	timer  *progressTimer
}

func (b *progressRequestBody) Read(p []byte) (int, error) {
	n, err := b.source.Read(p)
	if n > 0 {
		b.timer.touch()
	}
	return n, err
}

func (b *progressRequestBody) Close() error { return b.source.Close() }

type progressResponseBody struct {
	source io.ReadCloser
	timer  *progressTimer
	ctx    context.Context
	cancel context.CancelCauseFunc
}

func (b *progressResponseBody) Read(p []byte) (int, error) {
	n, err := b.source.Read(p)
	if n > 0 {
		b.timer.touch()
	}
	if err != nil && !errors.Is(err, io.EOF) {
		err = readStall(b.ctx, err)
	}
	return n, err
}

func (b *progressResponseBody) Close() error {
	b.timer.stop()
	err := b.source.Close()
	b.cancel(nil)
	return err
}

type progressTransport struct {
	next http.RoundTripper
	idle time.Duration
}

func (p progressTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancelCause(request.Context())
	timer := newProgressTimer(p.idle, cancel)
	request = request.WithContext(ctx)
	if request.Body != nil && request.Body != http.NoBody {
		request.Body = &progressRequestBody{source: request.Body, timer: timer}
	}
	response, err := p.next.RoundTrip(request)
	if err != nil {
		timer.stop()
		stalled := readStall(ctx, err)
		cancel(nil)
		return nil, stalled
	}
	response.Body = &progressResponseBody{source: response.Body, timer: timer, ctx: ctx, cancel: cancel}
	return response, nil
}
