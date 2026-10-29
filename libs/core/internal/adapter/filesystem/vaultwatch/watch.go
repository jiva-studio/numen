package vaultwatch

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/rjeczalik/notify"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/watchhub"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Backlog is how many events are held while they are being debounced. Past it the
// vault is read from scratch, which is cheaper than working out what was
// missed. A checkout of a large repository fits inside it; an archive unpacked
// over a whole vault does not, and that is the case the bound is for.
const Backlog = 65536

// handover is how many events the channel the watcher writes into can hold. A
// scheduling; the bound that decides anything is Backlog.
const handover = 1024

// Watcher follows vaults on disk.
type Watcher struct {
	Options vaultconfig.Options
}

// WatchVault follows one vault and reports the sources that change in it.
//
// The watch is on the tree, not on any file: editors save by writing a
// temporary file and renaming it over the original, so the file a watch was
// placed on stops existing at the first save.
func (w Watcher) WatchVault(
	ctx context.Context,
	v domain.Vault,
) (changes <-chan []string, lostChan <-chan struct{}, err error) {
	reader, err := vaultreader.Open(v.Path, w.Options)
	if err != nil {
		return nil, nil, err
	}
	opts := w.Options

	// The shape is taken before the first event, because the first event may be
	// a folder leaving: what it was can only be known from before it went.
	shape, why := vaultreader.ReadShape(reader)

	raw := make(chan notify.EventInfo, handover)
	stop, err := watchhub.Watch(reader.Root(), true, raw)
	if err != nil {
		return nil, nil, err
	}

	debounced := make(chan []string)
	removed := make(chan struct{}, 1)

	// A shape short of a folder the walk could not enter cannot tell a folder
	// that has gone from a file that has, so the vault is read again.
	if why != nil {
		removed <- struct{}{}
	}

	go func() {
		defer stop()
		defer close(debounced)
		queue := newQueue(Backlog)
		go forwardEvents(ctx, raw, queue)
		debounce(ctx, shape, opts, queue, debounced, removed)
	}()

	return debounced, removed, nil
}

// File follows one file and reports each time it changes.
//
// The watch is on the folder and not on the file: a file is replaced by being
// written beside itself and renamed over, so a watch placed on the file itself
// stops existing at the first write. Everything under that name is one file
// here — a database and the journals beside it are written together and are one
// change.
func (w Watcher) File(ctx context.Context, path string) (<-chan struct{}, error) {
	pos, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(pos)

	raw := make(chan notify.EventInfo, handover)
	stop, err := watchhub.Watch(filepath.Dir(pos), false, raw)
	if err != nil {
		return nil, err
	}

	moved := make(chan struct{}, 1)
	go relayFileEvents(ctx, raw, stop, name, moved)
	return moved, nil
}

// relayFileEvents reports each event under the file's name until the context
// ends, then stops the watch and closes the report. At most one message is
// pending: a listener that has not read the last one will query anyway.
func relayFileEvents(ctx context.Context, raw chan notify.EventInfo, stop func(), name string, moved chan<- struct{}) {
	defer close(moved)
	defer stop()
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
			select {
			case moved <- struct{}{}:
			default:
			}
		}
	}
}
