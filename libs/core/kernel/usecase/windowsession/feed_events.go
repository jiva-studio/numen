package windowsession

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// FeedQuietPeriod is how long the feed of a vault has to have been still before the plugins are told of its changes.
const FeedQuietPeriod = 500 * time.Millisecond

// FeedEvents tells the plugins that the file change feed of a vault holds changes they have not been told of.
// WatchWrites calls report after each committed write of the index that can change the feed of the vault,
// until the function it returns is called. WatchPlugins reports each time the plugin host applied settings
// or forgot the failure of a plugin, and the function it returns stops the reports. ReadHead returns the
// head of the feed, and Emit sends the event of the vault. Writes closer together than Quiet are one event.
type FeedEvents struct {
	WatchWrites  func(id domain.VaultID, report func()) func()
	WatchPlugins func() (<-chan struct{}, func())
	ReadHead     func(ctx context.Context, id domain.VaultID) (filechange.Head, error)
	Emit         func(v domain.Vault)
	Quiet        time.Duration
	After        func(time.Duration) <-chan time.Time
}

// start begins watching the feed of the vault and returns the loop that emits its events, or nil where the
// window emits none. The watch starts before the passes of the vault write anything. The loop emits once
// for a feed that holds changes when the vault opens, once after each burst of writes that moved the head
// of the feed, and once after each change of the plugins.
func (f FeedEvents) start(ctx context.Context, v domain.Vault) func() {
	if f.WatchWrites == nil || f.ReadHead == nil || f.Emit == nil {
		return nil
	}
	wake := make(chan struct{}, 1)
	stopWrites := f.WatchWrites(v.ID, func() { signal(wake) })
	signal(wake)
	return func() {
		defer stopWrites()
		var plugins <-chan struct{}
		if f.WatchPlugins != nil {
			changes, stop := f.WatchPlugins()
			defer stop()
			plugins = changes
		}
		var told filechange.Cursor
		for {
			isForced := false
			select {
			case <-ctx.Done():
				return
			case <-wake:
			case <-plugins:
				isForced = true
			}
			select {
			case <-ctx.Done():
				return
			case <-f.after(f.Quiet):
			}
			head, err := f.ReadHead(ctx, v.ID)
			if err != nil || head.Cursor.Seq == 0 {
				continue
			}
			if isForced || head.Cursor != told {
				told = head.Cursor
				f.Emit(v)
			}
		}
	}
}

func (f FeedEvents) after(d time.Duration) <-chan time.Time {
	if f.After != nil {
		return f.After(d)
	}
	return time.After(d)
}
