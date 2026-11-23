package feed

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// SettingAutomatic is the setting that turns on Recognize automatically.
const SettingAutomatic = "recognizeAutomatically"

// Automatic recognizes the scans the file change feed of a vault names while Recognize automatically is on: on each
// onFilesChanged event of the vault, and over the shown vault when the setting is first read on and
// when it is turned on.
type Automatic struct {
	follower      *Follower
	getShownVault func(ctx context.Context) (domain.VaultID, error)
	report        func(what string, err error)
	isOn          atomic.Bool
	closing       chan struct{}
	wg            sync.WaitGroup
}

// NewAutomatic returns the automatic runs of the follower, over the vault shown reads when the setting
// turns on. report receives the failures of the background work.
func NewAutomatic(
	follower *Follower, shown func(ctx context.Context) (domain.VaultID, error), report func(what string, err error),
) *Automatic {
	return &Automatic{follower: follower, getShownVault: shown, report: report, isOn: atomic.Bool{}, closing: make(chan struct{}), wg: sync.WaitGroup{}}
}

// UpdateSettings reads SettingAutomatic from values, and follows the feed of the shown vault in the
// background when it turned on.
func (a *Automatic) UpdateSettings(values map[string]string) {
	isOn := values[SettingAutomatic] == "true"
	if wasOn := a.isOn.Swap(isOn); wasOn || !isOn {
		return
	}
	a.wg.Go(func() {
		ctx, cancel := a.withClosing()
		defer cancel()
		vault, err := a.getShownVault(ctx)
		if err != nil {
			return
		}
		if err := a.follower.Execute(ctx, vault); err != nil && ctx.Err() == nil {
			a.report("follow the file changes of "+string(vault), err)
		}
	})
}

// HandleEvent follows the feed of the vault an onFilesChanged event names while the setting is on, and
// ignores any other event.
func (a *Automatic) HandleEvent(ctx context.Context, event string) error {
	vault, ok := host.ReadFilesChangedEvent(event)
	if !ok || !a.isOn.Load() {
		return nil
	}
	return a.follower.Execute(ctx, vault)
}

func (a *Automatic) withClosing() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-a.closing:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

// Close stops the background work and waits for it to end.
func (a *Automatic) Close() {
	close(a.closing)
	a.wg.Wait()
}
