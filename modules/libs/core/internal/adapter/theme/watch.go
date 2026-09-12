package theme

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/rjeczalik/notify"
)

// ErrNoFolder is a catalogue with nowhere to read the person's themes from.
var ErrNoFolder = errors.New("there is no themes folder")

// Backlog is how many events are held while they are being debounced. The folder
// holds a handful of files.
const Backlog = 256

// DefaultHold is how long changes are kept before they are reported. An editor
// saves a file by writing a temporary one and renaming it over the top, and
// that is several events.
const DefaultHold = 50 * time.Millisecond

// Watch reports the person's themes changing, by name, until ctx ends.
//
// The watch is on the folder and not on any file: the file a watch was placed
// on stops existing at the first save. One level, which is as deep as a theme
// is.
func (c Catalogue) Watch(ctx context.Context, hold time.Duration) (<-chan []string, error) {
	if c.dir == "" {
		return nil, ErrNoFolder
	}
	if hold <= 0 {
		hold = DefaultHold
	}

	raw := make(chan notify.EventInfo, Backlog)
	if err := notify.Watch(c.dir, raw, notify.All); err != nil {
		return nil, fmt.Errorf("watch %s: %w", c.dir, err)
	}

	debounced := make(chan []string)
	go func() {
		defer notify.Stop(raw)
		defer close(debounced)
		c.debounce(ctx, hold, raw, debounced)
	}()
	return debounced, nil
}

// debounce collects events for a hold and reports each theme once.
//
// Reading the events and delivering them are kept apart: whoever listens takes
// as long as it takes, and the operating system goes on producing events
// meanwhile.
func (c Catalogue) debounce(
	ctx context.Context,
	hold time.Duration,
	raw <-chan notify.EventInfo,
	changed chan<- []string,
) {
	pending := map[string]bool{}
	// ready is what a hold has already closed over, waiting to be taken. It
	// keeps growing while nobody takes it: one batch, never two.
	var ready []string
	inReady := map[string]bool{}
	var held <-chan time.Time

	for {
		// The delivery arm is only in the select when there is something to
		// deliver; a nil channel is never chosen.
		var out chan<- []string
		if len(ready) > 0 {
			out = changed
		}

		select {
		case <-ctx.Done():
			return

		case out <- ready:
			ready = nil
			clear(inReady)

		case event, open := <-raw:
			if !open {
				return
			}
			name, is := c.names(event.Path())
			if !is {
				continue
			}
			pending[name] = true
			// Measured from the first event of a batch, so a folder being
			// written to continuously is still reported.
			if held == nil {
				held = time.After(hold)
			}

		case <-held:
			held = nil
			for name := range pending {
				if !inReady[name] {
					inReady[name] = true
					ready = append(ready, name)
				}
			}
			clear(pending)
		}
	}
}

// names is the theme an event is about, where it is about one. A file one level
// down is not a theme and neither is a name that does not end `.css`.
func (c Catalogue) names(absolute string) (string, bool) {
	if filepath.Clean(filepath.Dir(absolute)) != filepath.Clean(c.dir) {
		return "", false
	}
	filename := filepath.Base(absolute)
	if !strings.HasSuffix(filename, Extension) {
		return "", false
	}
	title := strings.TrimSuffix(filename, Extension)
	if !plain(title) {
		return "", false
	}
	return string(Mine) + ":" + title, true
}
