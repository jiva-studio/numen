package recognition

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// Provider is the plugin's text provider: the recognition of a file as the host reads it, a
// recognition started for the host, and one deleted. A started recognition runs until it ends or the
// provider is closed.
type Provider struct {
	recognition *Recognition
	store       host.Store

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.Mutex
	running map[string]bool
}

var _ host.TextProvider = (*Provider)(nil)

// NewProvider is the provider over the recognition, keeping its texts in the store.
func NewProvider(recognition *Recognition, store host.Store) *Provider {
	ctx, cancel := context.WithCancel(context.Background())
	return &Provider{recognition: recognition, store: store, ctx: ctx, cancel: cancel, running: map[string]bool{}}
}

// Wait waits for the recognitions it started to end.
func (p *Provider) Wait() { p.wg.Wait() }

// Close stops the recognitions it started and waits for them to end.
func (p *Provider) Close() {
	p.cancel()
	p.Wait()
}

// GetText is one page of the file's recognition a call, the cursor naming the page.
func (p *Provider) GetText(ctx context.Context, req host.TextRequest) (host.TextPage, error) {
	if req.Key != VersionKey {
		return host.TextPage{}, fmt.Errorf("key %q: %w", req.Key, host.ErrNotExist)
	}
	kept, err := readRecord(ctx, p.getStore(req.Vault), req.Path)
	if err != nil {
		return host.TextPage{}, err
	}
	if kept == nil || !kept.IsComplete {
		return host.TextPage{}, fmt.Errorf("the recognition of %s: %w", req.Path, host.ErrNotExist)
	}
	index := 0
	if req.Cursor != "" {
		index, err = strconv.Atoi(req.Cursor)
		if err != nil || index < 0 || index >= len(kept.Pages) {
			return host.TextPage{}, fmt.Errorf("cursor %q of %s: %w", req.Cursor, req.Path, host.ErrNotExist)
		}
	}
	if len(kept.Pages) == 0 {
		return host.TextPage{}, nil
	}
	page := host.TextPage{Parts: kept.getTextParts(index)}
	if index+1 < len(kept.Pages) {
		page.NextCursor = strconv.Itoa(index + 1)
	}
	return page, nil
}

// CreateText starts recognising the file and returns. The plugin reads files, so a request for text
// of a version is invalid. A file being recognised already is not started again.
func (p *Provider) CreateText(_ context.Context, req host.TextCreation) error {
	if req.Input != "" {
		return fmt.Errorf("the plugin recognises files and not %s: %w", req.Input, host.ErrInvalidVersion)
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

// DeleteText removes the file's recognition.
func (p *Provider) DeleteText(ctx context.Context, vault domain.VaultID, path, key string) error {
	if key != VersionKey {
		return nil
	}
	return p.getStore(vault).Delete(ctx, getRecordName(path))
}

// getStore is the plugin's folder in the vault, the shown one where the vault is empty.
func (p *Provider) getStore(vault domain.VaultID) host.Store {
	if vault == "" {
		return p.store
	}
	return p.store.With(vault)
}
