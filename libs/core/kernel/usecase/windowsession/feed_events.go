package windowsession

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// FeedQuietPeriod is how long the feed of a vault has to have been still before the plugins are notified
// of its changes.
const FeedQuietPeriod = 500 * time.Millisecond

// FeedEvents notifies the plugins that the file change feed of a vault holds changes they have not been
// notified of. WatchWrites calls report after each committed write of the index that can change the feed
// of the vault, until the function it returns is called. WatchPlugins reports each time the plugin host
// applied settings or forgot the failure of a plugin, and the function it returns stops the reports.
// ReadHead returns the head of the feed, and Emit sends the event of the vault. Writes closer together
// than Quiet are one event. The watch starts before the passes of the vault write anything, and the event
// is sent once for a feed that holds changes when the vault opens, once after each burst of writes that
// moved the head of the feed, and once after each change of the plugins. A window without plugins sends
// none.
type FeedEvents struct {
	WatchWrites  func(id domain.VaultID, report func()) func()
	WatchPlugins func() (<-chan struct{}, func())
	ReadHead     func(ctx context.Context, id domain.VaultID) (filechange.Head, error)
	Emit         func(v domain.Vault)
	Quiet        time.Duration
	After        func(time.Duration) <-chan time.Time
}

func (f FeedEvents) start(ctx context.Context, v domain.Vault) func() {
	if f.WatchWrites == nil || f.ReadHead == nil || f.Emit == nil {
		return nil
	}
	loop := &feedLoop{events: f, vault: v, wake: make(chan struct{}, 1), plugins: nil, sent: filechange.Cursor{}}
	stopWrites := f.WatchWrites(v.ID, func() { signal(loop.wake) })
	signal(loop.wake)
	return func() {
		defer stopWrites()
		if f.WatchPlugins != nil {
			changes, stop := f.WatchPlugins()
			defer stop()
			loop.plugins = changes
		}
		loop.run(ctx)
	}
}

// feedLoop is the loop that sends the events of one vault. sent is the head of the feed at the last event.
type feedLoop struct {
	events  FeedEvents
	vault   domain.Vault
	wake    chan struct{}
	plugins <-chan struct{}
	sent    filechange.Cursor
}

func (l *feedLoop) run(ctx context.Context) {
	for {
		isForced, ok := l.waitForChange(ctx)
		if !ok {
			return
		}
		head, err := l.events.ReadHead(ctx, l.vault.ID)
		if err != nil || head.Cursor.Seq == 0 {
			continue
		}
		if isForced || head.Cursor != l.sent {
			l.sent = head.Cursor
			l.events.Emit(l.vault)
		}
	}
}

func (l *feedLoop) waitForChange(ctx context.Context) (isForced, ok bool) {
	select {
	case <-ctx.Done():
		return false, false
	case <-l.wake:
	case <-l.plugins:
		isForced = true
	}
	select {
	case <-ctx.Done():
		return false, false
	case <-l.getQuiet():
	}
	return isForced, true
}

func (l *feedLoop) getQuiet() <-chan time.Time {
	if l.events.After != nil {
		return l.events.After(l.events.Quiet)
	}
	return time.After(l.events.Quiet)
}
