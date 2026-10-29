package recognition

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// Provider is the plugin's text provider: a recognition started for the host. A started recognition
// runs until it ends or the provider is closed.
type Provider struct {
	recognition *Recognition

	wg sync.WaitGroup

	mu       sync.Mutex
	isClosed bool
	running  map[string]context.CancelFunc
}

var _ host.TextProvider = (*Provider)(nil)

// NewProvider is the provider over the recognition.
func NewProvider(recognition *Recognition) *Provider {
	return &Provider{
		recognition: recognition,
		wg:          sync.WaitGroup{},
		mu:          sync.Mutex{},
		isClosed:    false,
		running:     map[string]context.CancelFunc{},
	}
}

// Wait waits for the recognitions it started to end.
func (p *Provider) Wait() { p.wg.Wait() }

// Close stops the recognitions it started and waits for them to end.
func (p *Provider) Close() {
	p.mu.Lock()
	p.isClosed = true
	for _, cancel := range p.running {
		cancel()
	}
	p.mu.Unlock()
	p.Wait()
}

// CreateText starts recognising the file and returns. A file being recognised already is not
// started again.
func (p *Provider) CreateText(ctx context.Context, req host.TextCreation) error {
	key := string(req.Vault) + "\x00" + req.Path
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
		// Recognise reports a failure as a task error.
		_, _ = p.recognition.Recognise(runCtx, Request{Vault: req.Vault, Path: req.Path, IsUserStarted: false})
	})
	return nil
}

func (p *Provider) finish(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.running, key)
}
