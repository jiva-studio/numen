package theme

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/rjeczalik/notify"
)

// ErrNoFolder is returned by a catalog with no folder to read user themes from.
var ErrNoFolder = errors.New("there is no themes folder")

// Backlog is how many events are held while they are being debounced. The folder
// holds a handful of files.
const Backlog = 256

// DefaultDebounceDelay is how long changes are kept before they are reported. An editor
// saves a file by writing a temporary one and renaming it over the top, and
// that is several events.
const DefaultDebounceDelay = 50 * time.Millisecond

// Watch reports changes to the user's themes, by name, until ctx ends.
//
// The watch is on the folder, not on a file: a watch on a file ends at the first
// save, which replaces the file. The watch is one level deep, which is as deep as
// a theme is.
func (c Catalog) Watch(ctx context.Context, hold time.Duration) (<-chan []string, error) {
	if c.dir == "" {
		return nil, ErrNoFolder
	}
	if hold <= 0 {
		hold = DefaultDebounceDelay
	}

	raw := make(chan notify.EventInfo, Backlog)
	stop, err := c.hub.Watch(c.dir, false, raw)
	if err != nil {
		return nil, err
	}

	debounced := make(chan []string)
	go func() {
		defer stop()
		defer close(debounced)
		c.debounce(ctx, hold, raw, debounced)
	}()
	return debounced, nil
}

func (c Catalog) debounce(
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
			name, is := c.getThemeName(event.Path())
			if !is {
				continue
			}
			batch.pending[name] = true
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

// themeBatch is the themes an event named, and the themes a hold has closed
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

func (c Catalog) getThemeName(absolute string) (string, bool) {
	if filepath.Clean(filepath.Dir(absolute)) != filepath.Clean(c.dir) {
		return "", false
	}
	filename := filepath.Base(absolute)
	if !strings.HasSuffix(filename, Extension) {
		return "", false
	}
	title := strings.TrimSuffix(filename, Extension)
	if !isPlainName(title) {
		return "", false
	}
	return string(Mine) + ":" + title, true
}
