package vault

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Follow keeps the index level with a vault that is being edited underneath it,
// and says which notes moved.
//
// The rule it carries is not about any one way of showing a vault: what
// changed is reindexed, and when what changed cannot be worked out the vault is
// read again. A window and a command line want the same thing to happen.
type Follow struct {
	Watcher port.VaultWatcher
	Refresh Refresh
	Scan    Scan

	// Changed, if set, is called each time the index and the vault are level
	// again. What is done with that is the caller's business.
	Changed func(Moved)
	// Trouble, if set, is called with what went wrong, and with nil when a
	// later attempt succeeds. Both, so what is reported is the state of things
	// now.
	Trouble func(error)
}

// Moved is what a caller is told: the notes that are different now, or that the
// whole vault has to be looked at again.
type Moved struct {
	Paths  []string
	Reload bool
	// Assets is the paths of the files that changed and are not notes. Reading
	// one is its own work and takes minutes. A reload carries none, and stands
	// for every asset in the vault.
	Assets []string
}

// Reading says whether an asset owes a read: one changed, or the whole vault is
// being looked at again and every asset with it.
func (m Moved) Reading() bool { return m.Reload || len(m.Assets) > 0 }

// Begin starts watching. Acting on what it collects is Run, and the two are
// separate because they belong at different moments.
//
// The watch belongs before the first scan: an edit made while the vault is
// being read is then held. Reading the vault again waits its turn behind a walk
// already running.
func (u Follow) Begin(ctx context.Context, v domain.Vault) (*Following, error) {
	changes, lost, err := u.Watcher.Watch(ctx, v)
	if err != nil {
		return nil, err
	}
	return &Following{follow: u, vault: v, changes: changes, lost: lost}, nil
}

// Following is a vault being watched, whose events are not being acted on yet.
type Following struct {
	follow  Follow
	vault   domain.Vault
	changes <-chan []string
	lost    <-chan struct{}
}

// Run acts on everything the watch has collected, and goes on until ctx is
// done or the watch stops.
func (f *Following) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case paths, watching := <-f.changes:
			if !watching {
				return
			}
			res, err := f.follow.Refresh.Execute(ctx, f.vault, paths)
			if err != nil {
				f.trouble(err)
				continue
			}
			f.trouble(nil)
			f.changed(Moved{Paths: res.Changed(), Assets: res.Assets})

		case <-f.lost:
			// More changed at once than could be followed, or something went
			// that cannot be asked what it held. Reading the vault again is the
			// answer, and whoever is listening is told to ask again.
			if _, err := f.follow.Scan.Execute(ctx, f.vault); err != nil {
				f.trouble(err)
				continue
			}
			f.trouble(nil)
			// Read again from the top, so whatever changed is among what the
			// walk finds.
			f.changed(Moved{Reload: true})
		}
	}
}

func (f *Following) changed(m Moved) {
	if f.follow.Changed != nil {
		f.follow.Changed(m)
	}
}

func (f *Following) trouble(err error) {
	if f.follow.Trouble != nil {
		f.follow.Trouble(err)
	}
}
