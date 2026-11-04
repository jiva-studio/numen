package service

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// modelOpening is one load of the local model, which runs on its own goroutine so that a download
// holds no lock a call of the host takes.
type modelOpening struct {
	ready  chan struct{}
	cancel context.CancelFunc
	emb    textEmbedder
	err    error
}

// errModelSuperseded ends an opening whose model the settings changed while it ran.
var errModelSuperseded = errors.New("the model was changed while it opened")

// getEmbedder returns the open model, and opens it when none is. The open runs apart from the
// caller: a caller that gives up leaves it running, and a call that only reads the model info or
// applies a setting never waits for it.
func (s *embeddingServer) getEmbedder(ctx context.Context) (textEmbedder, error) {
	for {
		emb, err := s.waitForOpening(ctx)
		if !errors.Is(err, errModelSuperseded) {
			return emb, err
		}
	}
}

func (s *embeddingServer) waitForOpening(ctx context.Context) (textEmbedder, error) {
	s.mu.Lock()
	if s.embedder != nil {
		emb := s.embedder
		s.mu.Unlock()
		return emb, nil
	}
	op := s.opening
	if op == nil {
		op = s.startOpeningLocked(ctx)
	}
	s.mu.Unlock()

	select {
	case <-op.ready:
		return op.emb, op.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// startOpeningLocked starts loading the model with the settings in force.
func (s *embeddingServer) startOpeningLocked(caller context.Context) *modelOpening {
	localInfo := s.modelInfo
	if !s.isMock {
		localInfo = s.localInfo
	}
	local := s.localModel
	limits := s.getLimitsLocked()
	if local.Threads == 0 {
		local.Threads = limits.threads
	}
	local.BatchTexts = limits.batchTexts

	// The load outlives the call that started it, and ends with the settings or the plugin.
	ctx, cancel := context.WithCancel(context.WithoutCancel(caller))
	op := &modelOpening{ready: make(chan struct{}), cancel: cancel, emb: nil, err: nil}
	s.opening = op
	open := s.open
	go func() {
		defer cancel()
		emb, err := open(ctx, localInfo, local)
		s.finishOpening(op, emb, err, limits)
	}()
	return op
}

// finishOpening records the model an opening loaded, unless the settings retired the opening.
func (s *embeddingServer) finishOpening(op *modelOpening, emb textEmbedder, err error, limits runLimits) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer close(op.ready)

	if s.opening != op {
		op.err = errModelSuperseded
		if err == nil {
			s.closeRetired(emb)
		}
		return
	}
	s.opening = nil
	if err != nil {
		op.err = fmt.Errorf("opening onnx embedding engine: %w", err)
		return
	}
	// The runtime starts its threads while it opens the model, and a thread starts at the niceness of
	// the thread that starts it.
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr log on priority error has no side effect"
	if err := s.lowerNice(limits.nice); err != nil {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="stderr log on priority error has no side effect"
		fmt.Fprintf(os.Stderr, "set the priority of the embeddings plugin: %v\n", err)
	}
	s.embedder = emb
	s.modelInfo.Concurrency = emb.Concurrency()
	op.emb = emb
}

// retireOpeningLocked drops a model that is loading, so that the next batch loads the model the
// settings name now.
func (s *embeddingServer) retireOpeningLocked() {
	if s.opening != nil {
		s.opening.cancel()
		s.opening = nil
	}
}

func (s *embeddingServer) closeRetired(emb textEmbedder) {
	go func() {
		// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr log on background close error has no side effect"
		if err := emb.Close(); err != nil {
			// gomutants:disable-next-line STATEMENT_REMOVE reason="stderr log on background close error has no side effect"
			fmt.Fprintf(os.Stderr, "close the embedder: %v\n", err)
		}
	}()
}

func (s *embeddingServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.retireOpeningLocked()
	if s.embedder != nil {
		err := s.embedder.Close()
		s.embedder = nil
		return err
	}
	return nil
}
