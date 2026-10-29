package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// provider starts a proofreading for the host. A started proofreading runs until it ends or the
// provider is closed.
type provider struct {
	proofreading *Proofreading

	wg sync.WaitGroup

	mu       sync.Mutex
	isClosed bool
	running  map[string]context.CancelFunc
}

var _ host.TextProvider = (*provider)(nil)

func newProvider(proofreading *Proofreading) *provider {
	return &provider{
		proofreading: proofreading, wg: sync.WaitGroup{},
		mu: sync.Mutex{}, isClosed: false, running: map[string]context.CancelFunc{},
	}
}

// Wait waits for the proofreadings it started to end.
func (p *provider) Wait() { p.wg.Wait() }

// Close stops the proofreadings it started and waits for them to end.
func (p *provider) Close() {
	p.mu.Lock()
	p.isClosed = true
	for _, cancel := range p.running {
		cancel()
	}
	p.mu.Unlock()
	p.Wait()
}

// CreateText starts proofreading the version the request names and returns. The plugin reads
// versions, so a request for text of the file itself is invalid. A version being proofread already
// is not started again.
func (p *provider) CreateText(ctx context.Context, req host.TextCreation) error {
	if req.Source == "" {
		return fmt.Errorf("the plugin proofreads versions and not the file %s: %w", req.Path, host.ErrInvalidVersion)
	}
	key := string(req.Vault) + "\x00" + req.Path + "\x00" + string(req.Source)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.isClosed || p.running[key] != nil {
		return nil
	}
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	p.running[key] = cancel
	p.wg.Go(func() {
		defer p.finish(key)
		defer cancel()
		// Proofread reports a failure as a task error.
		_, _ = p.proofreading.Proofread(runCtx, Request{
			Vault: req.Vault, Path: req.Path, Source: req.Source, IsUserStarted: req.IsUserStarted,
		})
	})
	return nil
}

func (p *provider) finish(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.running, key)
}

// newProofreadCommand proofreads the live text of the file the user named and waits for the result.
func newProofreadCommand(proofreading *Proofreading) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		res, err := proofreading.Proofread(ctx, Request{Vault: "", Path: req.Path, Source: "", IsUserStarted: true})
		if err != nil {
			return command.Result{}, err
		}
		message := fmt.Sprintf("proofread %d lines of %s and put %d right", res.Lines, req.Path, res.Corrected)
		if res.Uncorrected > 0 {
			message += fmt.Sprintf("; the model gave no answer for %d batches", res.Uncorrected)
		}
		return command.Result{Message: message, OpenedPath: ""}, nil
	}
}
