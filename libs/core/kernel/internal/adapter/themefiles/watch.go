package themefiles

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/rjeczalik/notify"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
)

// Backlog is how many events are held while they are being debounced. The folder
// holds a handful of files.
const Backlog = 256

// DefaultDebounceDelay is how long changes are kept before they are reported. An editor
// saves a file by writing a temporary one and renaming it over the top, and
// that is several events.
const DefaultDebounceDelay = 50 * time.Millisecond

// WatchOwnThemes reports changes to the user's themes, by title, until ctx ends.
//
// The watch is on the folder, not on a file: a watch on a file ends at the first
// save, which replaces the file. The watch is one level deep, which is as deep as
// a theme is.
func (f Files) WatchOwnThemes(ctx context.Context, hold time.Duration) (<-chan []string, error) {
	if f.dir == "" {
		return nil, themedomain.ErrNoFolder
	}
	if hold <= 0 {
		hold = DefaultDebounceDelay
	}

	raw := make(chan notify.EventInfo, Backlog)
	stop, err := f.hub.Watch(f.dir, false, raw)
	if err != nil {
		return nil, err
	}

	debounced := make(chan []string)
	go func() {
		defer stop()
		defer close(debounced)
		f.debounce(ctx, hold, raw, debounced)
	}()
	return debounced, nil
}

func (f Files) debounce(
	ctx context.Context,
	hold time.Duration,
	raw <-chan notify.EventInfo,
	changed chan<- []string,
) {
	batch := newThemeBatch()
	var target <-chan time.Time

	for {
		// The delivery arm is only in the select when there is something to
		// deliver; a nil channel is never chosen.
		var out chan<- []string
		if len(batch.ready) > 0 {
			out = changed
		}

		select {
		case <-ctx.Done():
			return

		case out <- batch.ready:
			batch.clearReady()

		case event, open := <-raw:
			if !open {
				return
			}
			title, is := f.getTitle(event.Path())
			if !is {
				continue
			}
			batch.pending[title] = true
			// Measured from the first event of a batch, so a folder being
			// written to continuously is still reported.
			if target == nil {
				target = time.After(hold)
			}

		case <-target:
			target = nil
			batch.closeHold()
		}
	}
}

// themeBatch is the titles an event named, and the themes a hold has closed
// over.
type themeBatch struct {
	pending map[string]bool
	ready   []string
	inReady map[string]bool
}

func newThemeBatch() *themeBatch {
	return &themeBatch{pending: map[string]bool{}, inReady: map[string]bool{}}
}

func (b *themeBatch) clearReady() {
	b.ready = nil
	clear(b.inReady)
}

func (b *themeBatch) closeHold() {
	for name := range b.pending {
		if !b.inReady[name] {
			b.inReady[name] = true
			b.ready = append(b.ready, name)
		}
	}
	clear(b.pending)
}

func (f Files) getTitle(absolute string) (string, bool) {
	if filepath.Clean(filepath.Dir(absolute)) != filepath.Clean(f.dir) {
		return "", false
	}
	filename := filepath.Base(absolute)
	if !strings.HasSuffix(filename, themedomain.Extension) {
		return "", false
	}
	title := strings.TrimSuffix(filename, themedomain.Extension)
	if !themedomain.IsPlainTitle(title) {
		return "", false
	}
	return title, true
}
