package windowsession

import (
	"context"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// FeedQuietPeriod is how long the feed of a vault has to have been still before the plugins are notified
// of its changes.
const FeedQuietPeriod = 500 * time.Millisecond

// FeedEvents notifies the plugins that the file change feed of a vault holds changes they have not been
// notified of. WatchWrites calls report after each committed write of the index that can change the feed
// of the vault, until the function it returns is called. WatchPlugins reports each time the plugin host
// applied settings or forgot the failure of a plugin, and the function it returns stops the reports.
// ReadHead returns the head of the feed, ListFiles the paths of the files of the changes after one number
// and up to another, and GetContentType the type of a file by its path. Emit sends the event of the vault
// with the types of the files the event covers. Writes closer together than Quiet are one event. The watch
// starts from the head the feed holds when the vault opens, before the passes of the vault write anything,
// and the event is sent once after each burst of writes that moved the head of the feed, with the types of
// the files those writes changed, and once after each change of the plugins, with the types of every file
// of the feed. A read of the feed that failed is tried again once, after the next quiet period. A window
// without plugins sends none.
type FeedEvents struct {
	WatchWrites    func(id domain.VaultID, report func()) func()
	WatchPlugins   func() (<-chan struct{}, func())
	ReadHead       func(ctx context.Context, id domain.VaultID) (filechange.Head, error)
	ListFiles      func(ctx context.Context, id domain.VaultID, after, upTo int64) ([]filechange.File, error)
	GetContentType func(path string) domain.ContentType
	Emit           func(v domain.Vault, types []domain.FileType)
	Quiet          time.Duration
	CreateTimer    func(time.Duration) <-chan time.Time
}

// Start begins the watch of the vault and returns the loop that sends its events until ctx ends, or nil
// where a part is missing.
func (f FeedEvents) Start(ctx context.Context, v domain.Vault) func() {
	if f.WatchWrites == nil || f.ReadHead == nil || f.ListFiles == nil || f.GetContentType == nil || f.Emit == nil {
		return nil
	}
	var initial filechange.Cursor
	if head, err := f.ReadHead(ctx, v.ID); err == nil {
		initial = head.Cursor
	}
	loop := &feedLoop{events: f, vault: v, wake: make(chan struct{}, 1), plugins: nil, sent: initial}
	stopWrites := f.WatchWrites(v.ID, func() { signal(loop.wake) })
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
	retry := notRetried
	for {
		isForced, ok := l.waitForChange(ctx)
		if !ok {
			return
		}
		if retry == retriedForced {
			isForced = true
			retry = retriedPlain
		}
		if err := l.send(ctx, isForced); err != nil {
			retry = l.fail(retry, isForced)
			continue
		}
		retry = notRetried
	}
}

func (l *feedLoop) send(ctx context.Context, isForced bool) error {
	head, err := l.events.ReadHead(ctx, l.vault.ID)
	if err != nil {
		return err
	}
	if head.Cursor.Seq == 0 || (!isForced && head.Cursor == l.sent) {
		return nil
	}
	after := l.sent.Seq
	if isForced || l.sent.FeedID != head.Cursor.FeedID {
		after = 0
	}
	types, err := l.listTypes(ctx, after, head.Cursor.Seq)
	if err != nil {
		return err
	}
	l.sent = head.Cursor
	l.events.Emit(l.vault, types)
	return nil
}

type retryState int

const (
	notRetried retryState = iota
	retriedPlain
	retriedForced
)

func (l *feedLoop) fail(retry retryState, isForced bool) retryState {
	if retry == notRetried {
		signal(l.wake)
	}
	if isForced {
		return retriedForced
	}
	return retriedPlain
}

func (l *feedLoop) listTypes(ctx context.Context, after, upTo int64) ([]domain.FileType, error) {
	files, err := l.events.ListFiles(ctx, l.vault.ID, after, upTo)
	if err != nil {
		return nil, err
	}
	types := make([]domain.FileType, 0, len(files))
	for _, file := range files {
		types = append(types, domain.FileType{Content: l.events.GetContentType(file.Path), Note: file.NoteType})
	}
	slices.SortFunc(types, domain.CompareFileTypes)
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
	if l.events.CreateTimer != nil {
		return l.events.CreateTimer(l.events.Quiet)
	}
	return time.After(l.events.Quiet)
}
