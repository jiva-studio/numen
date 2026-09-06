package filesystem

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rjeczalik/notify"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Backlog is how many events are held while they are being debounced. Past it the
// vault is read from scratch, which is cheaper than working out what was
// missed. A checkout of a large repository fits inside it; an archive unpacked
// over a whole vault does not, and that is the case the bound is for.
const Backlog = 65536

// handover is how many events the channel the watcher writes into can hold. A
// reader that does nothing else empties it, so what stands in it is a moment of
// scheduling; the bound that decides anything is Backlog.
const handover = 1024

// Watcher follows vaults on disk.
type Watcher struct {
	Options Options
}

// Watch follows one vault and reports the sources that change in it.
//
// The watch is on the tree, not on any file: editors save by writing a
// temporary file and renaming it over the original, so the file a watch was
// placed on stops existing at the first save.
func (w Watcher) Watch(
	ctx context.Context,
	v domain.Vault,
) (changes <-chan []string, lost <-chan struct{}, err error) {
	reader, err := Open(v.Path, w.Options)
	if err != nil {
		return nil, nil, err
	}
	opts := w.Options

	// The shape is taken before the first event, because the first event may be
	// a folder leaving: what it was can only be known from before it went.
	shape, why := remembered(reader)

	raw := make(chan notify.EventInfo, handover)
	tree := filepath.Join(reader.Root(), "...")
	if err := notify.Watch(tree, raw, notify.All); err != nil {
		return nil, nil, fmt.Errorf("watch %s: %w", v.Path, err)
	}

	debounced := make(chan []string)
	gone := make(chan struct{}, 1)

	// A shape short of a folder the walk could not enter cannot tell a folder
	// that has gone from a file that has, so the vault is read again.
	if why != nil {
		gone <- struct{}{}
	}

	go func() {
		defer notify.Stop(raw)
		defer close(debounced)
		waiting := newQueue(Backlog)
		go drain(ctx, raw, waiting)
		debounce(ctx, shape, opts, waiting, debounced, gone)
	}()

	return debounced, gone, nil
}

// File follows one file and reports each time it changes.
//
// The watch is on the folder and not on the file: a file is replaced by being
// written beside itself and renamed over, so a watch placed on the file itself
// stops existing at the first write. Everything under that name is one file
// here — a database and the journals beside it are written together and are one
// change.
func (w Watcher) File(ctx context.Context, path string) (<-chan struct{}, error) {
	at, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(at)

	raw := make(chan notify.EventInfo, handover)
	if err := notify.Watch(filepath.Dir(at), raw, notify.All); err != nil {
		return nil, fmt.Errorf("watch %s: %w", path, err)
	}

	moved := make(chan struct{}, 1)
	go func() {
		defer notify.Stop(raw)
		defer close(moved)
		for {
			select {
			case <-ctx.Done():
				return
			case event, open := <-raw:
				if !open {
					return
				}
				if !strings.HasPrefix(filepath.Base(event.Path()), name) {
					continue
				}
				// One waiting message is as much as this says: a listener that
				// has not read the last one is a listener about to ask anyway.
				select {
				case moved <- struct{}{}:
				default:
				}
			}
		}
	}()
	return moved, nil
}
