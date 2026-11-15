package windowsession

import (
	"context"
	"slices"
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
// ReadHead returns the head of the feed, ListPaths the paths of the files of the changes after one number
// and up to another, and GetContentType the type of a file by its path. Emit sends the event of the vault
// with the types of the files the event covers. Writes closer together than Quiet are one event. The watch
// starts before the passes of the vault write anything, and the event is sent once for a feed that holds
// changes when the vault opens, with the types of every file of the feed, once after each burst of writes
// that moved the head of the feed, with the types of the files those writes changed, and once after each
// change of the plugins, with the types of every file of the feed. A read of the feed that failed is tried
// again once, after the next quiet period. A window without plugins sends none.
type FeedEvents struct {
	WatchWrites    func(id domain.VaultID, report func()) func()
	WatchPlugins   func() (<-chan struct{}, func())
	ReadHead       func(ctx context.Context, id domain.VaultID) (filechange.Head, error)
	ListPaths      func(ctx context.Context, id domain.VaultID, after, upTo int64) ([]string, error)
	GetContentType func(path string) domain.ContentType
	Emit           func(v domain.Vault, types []domain.ContentType)
	Quiet          time.Duration
	After          func(time.Duration) <-chan time.Time
}

// Start begins the watch of the vault and returns the loop that sends its events until ctx ends, or nil
// where a part is missing.
func (f FeedEvents) Start(ctx context.Context, v domain.Vault) func() {
	if f.WatchWrites == nil || f.ReadHead == nil || f.ListPaths == nil || f.GetContentType == nil || f.Emit == nil {
		return nil
	}
	loop := &feedLoop{events: f, vault: v, wake: make(chan struct{}, 1), plugins: nil, sent: filechange.Cursor{}, retry: notRetried}
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
	retry   retryState
}

func (l *feedLoop) run(ctx context.Context) {
	for {
		isForced, ok := l.waitForChange(ctx)
		if !ok {
			return
		}
		if l.retry == retriedForced {
			isForced = true
			l.retry = retriedPlain
		}
		head, err := l.events.ReadHead(ctx, l.vault.ID)
		if err != nil {
			l.fail(isForced)
			continue
		}
		if head.Cursor.Seq == 0 || (!isForced && head.Cursor == l.sent) {
			l.retry = notRetried
			continue
		}
		after := l.sent.Seq
		if isForced || l.sent.FeedID != head.Cursor.FeedID {
			after = 0
		}
		types, err := l.listTypes(ctx, after, head.Cursor.Seq)
		if err != nil {
			l.fail(isForced)
			continue
		}
		l.retry = notRetried
		l.sent = head.Cursor
		l.events.Emit(l.vault, types)
	}
}

type retryState int

const (
	notRetried retryState = iota
	retriedPlain
	retriedForced
)

func (l *feedLoop) fail(isForced bool) {
	isFirst := l.retry == notRetried
	l.retry = retriedPlain
	if isForced {
		l.retry = retriedForced
	}
	if isFirst {
		signal(l.wake)
	}
}

func (l *feedLoop) listTypes(ctx context.Context, after, upTo int64) ([]domain.ContentType, error) {
	paths, err := l.events.ListPaths(ctx, l.vault.ID, after, upTo)
	if err != nil {
		return nil, err
	}
	types := make([]domain.ContentType, 0, len(paths))
	for _, path := range paths {
		types = append(types, l.events.GetContentType(path))
	}
	slices.Sort(types)
	return slices.Compact(types), nil
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
