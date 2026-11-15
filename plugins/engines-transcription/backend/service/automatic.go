package service

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/feed"
)

// settingAutomatic is the setting that turns on Transcribe automatically.
const settingAutomatic = "transcribeAutomatically"

// automatic transcribes the recordings the file change feed of a vault names while Transcribe automatically
// is on: on each onFilesChanged event of the vault, and over the shown vault when the setting is first
// read on and when it is turned on.
type automatic struct {
	follower *feed.Follower
	shown    func(ctx context.Context) (domain.VaultID, error)
	report   func(what string, err error)
	isOn     atomic.Bool
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func newAutomatic(
	follower *feed.Follower, shown func(ctx context.Context) (domain.VaultID, error), report func(what string, err error),
) *automatic {
	ctx, cancel := context.WithCancel(context.Background())
	return &automatic{follower: follower, shown: shown, report: report, isOn: atomic.Bool{}, ctx: ctx, cancel: cancel, wg: sync.WaitGroup{}}
}

// updateSettings reads the setting from values, and follows the feed of the shown vault in the
// background when it turned on.
func (a *automatic) updateSettings(values map[string]string) {
	isOn := values[settingAutomatic] == "true"
	if wasOn := a.isOn.Swap(isOn); wasOn || !isOn {
		return
	}
	a.wg.Go(func() {
		vault, err := a.shown(a.ctx)
		if err != nil {
			return
		}
		if err := a.follower.Execute(a.ctx, vault); err != nil && a.ctx.Err() == nil {
			a.report("follow the file changes of "+string(vault), err)
		}
	})
}

// handleEvent follows the feed of the vault an onFilesChanged event names while the setting is on,
// and ignores any other event.
func (a *automatic) handleEvent(ctx context.Context, event string) error {
	vault, ok := host.ReadFilesChangedEvent(event)
	if !ok || !a.isOn.Load() {
		return nil
	}
	return a.follower.Execute(ctx, vault)
}

func (a *automatic) close() {
	a.cancel()
	a.wg.Wait()
}
