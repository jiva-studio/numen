package recognition

import (
	"context"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// Provider is the plugin's text provider: a recognition started for the host. A started recognition
// runs until it ends or the provider is closed.
type Provider struct {
	recognition *Recognition

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.Mutex
	running map[string]bool
}

var _ host.TextProvider = (*Provider)(nil)

// NewProvider is the provider over the recognition.
func NewProvider(recognition *Recognition) *Provider {
	ctx, cancel := context.WithCancel(context.Background())
	return &Provider{recognition: recognition, ctx: ctx, cancel: cancel, running: map[string]bool{}}
}

// Wait waits for the recognitions it started to end.
func (p *Provider) Wait() { p.wg.Wait() }

// Close stops the recognitions it started and waits for them to end.
func (p *Provider) Close() {
	p.cancel()
	p.Wait()
}

// CreateText starts recognising the file and returns. The plugin reads files, so a request for text
// of a version is invalid. A file being recognised already is not started again.
func (p *Provider) CreateText(_ context.Context, req host.TextCreation) error {
	if req.Source != "" {
		return fmt.Errorf("the plugin recognises files and not %s: %w", req.Source, host.ErrInvalidVersion)
	}
	key := string(req.Vault) + "\x00" + req.Path
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running[key] {
		return nil
	}
	p.running[key] = true
	p.wg.Go(func() {
		defer p.finish(key)
		// Recognise reports a failure as a task error.
		_, _ = p.recognition.Recognise(p.ctx, Request{Vault: req.Vault, Path: req.Path, IsUserStarted: req.IsUserStarted})
	})
	return nil
}

func (p *Provider) finish(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.running, key)
}
