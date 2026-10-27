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

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.Mutex
	running map[string]bool
}

var _ host.TextProvider = (*provider)(nil)

func newProvider(proofreading *Proofreading) *provider {
	ctx, cancel := context.WithCancel(context.Background())
	return &provider{proofreading: proofreading, ctx: ctx, cancel: cancel, running: map[string]bool{}}
}

// Wait waits for the proofreadings it started to end.
func (p *provider) Wait() { p.wg.Wait() }

// Close stops the proofreadings it started and waits for them to end.
func (p *provider) Close() {
	p.cancel()
	p.Wait()
}

// CreateText starts proofreading the version the request names and returns. The plugin reads
// versions, so a request for text of the file itself is invalid. A version being proofread already
// is not started again.
func (p *provider) CreateText(_ context.Context, req host.TextCreation) error {
	if req.Source == "" {
		return fmt.Errorf("the plugin proofreads versions and not the file %s: %w", req.Path, host.ErrInvalidVersion)
	}
	key := string(req.Vault) + "\x00" + req.Path + "\x00" + string(req.Source)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running[key] {
		return nil
	}
	p.running[key] = true
	p.wg.Go(func() {
		defer p.finish(key)
		// Proofread reports a failure as a task error.
		_, _ = p.proofreading.Proofread(p.ctx, Request{
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

// newProofreadCommand proofreads the live text of the file a person named and waits for it.
func newProofreadCommand(proofreading *Proofreading) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		res, err := proofreading.Proofread(ctx, Request{Path: req.Path, IsUserStarted: true})
		if err != nil {
			return command.Result{}, err
		}
		message := fmt.Sprintf("proofread %d lines of %s and put %d right", res.Lines, req.Path, res.Corrected)
		if res.Uncorrected > 0 {
			message += fmt.Sprintf("; the model gave no answer for %d batches", res.Uncorrected)
		}
		return command.Result{Message: message}, nil
	}
}
