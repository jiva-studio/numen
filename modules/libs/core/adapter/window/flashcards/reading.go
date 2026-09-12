package flashcards

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// ReadVault is a vault walked into the index.
type ReadVault func(ctx context.Context, v domain.Vault) error

// readings is which vaults this window has read into the index: the ones read
// since it opened, the ones being read now, and why the last reading of one
// failed.
type readings struct {
	mu sync.Mutex

	// read is how a vault is brought up to date in the index, and under is the
	// life those readings run for.
	read  ReadVault
	under context.Context

	done     map[domain.VaultID]bool
	underway map[domain.VaultID]bool
	why      map[domain.VaultID]string
}

// on is how a vault is read from now on, and the life those readings run for.
func (r *readings) on(ctx context.Context, read ReadVault) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.under, r.read = ctx, read
}

// forget lets go of why a vault could not be read.
func (r *readings) forget(id domain.VaultID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.why, id)
}

// claim claims a vault for reading. It hands back the reader and the life to
// run it under where this call is the one that begins it, and no reader where
// the vault has been read, is being read, could not be read, or where this
// window reads nothing.
func (r *readings) claim(
	v domain.Vault,
) (read ReadVault, under context.Context, underway bool, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done[v.ID] {
		return nil, nil, false, ""
	}
	if why, told := r.why[v.ID]; told {
		return nil, nil, false, why
	}
	if r.underway[v.ID] {
		return nil, nil, true, ""
	}
	if r.read == nil {
		return nil, nil, false, ""
	}
	if r.underway == nil {
		r.underway = make(map[domain.VaultID]bool)
	}
	r.underway[v.ID] = true
	return r.read, r.under, true, ""
}

// finish is what one reading came to: the vault is read, or it is why it is not.
func (r *readings) finish(v domain.Vault, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.underway, v.ID)
	if err != nil {
		if r.why == nil {
			r.why = make(map[domain.VaultID]string)
		}
		r.why[v.ID] = err.Error()
		return
	}
	if r.done == nil {
		r.done = make(map[domain.VaultID]bool)
	}
	r.done[v.ID] = true
}

// Reading is how a vault is brought up to date in the index, and the life those
// readings run for. A window naming none counts a vault from the index as it
// stands.
func (a *API) Reading(ctx context.Context, read ReadVault) {
	a.readings.on(ctx, read)
}

// reading brings a vault up to date in the index. It says whether a reading of
// that vault is running now, and why the last one failed, when one did.
//
// A vault is read once for the life of the window, and the watcher carries it
// from there. One vault is read once at a time however many counts ask for it,
// and a reading that failed is not begun again until the vault moves.
func (a *API) reading(ctx context.Context, v domain.Vault) (underway bool, reason string) {
	read, behind, running, reason := a.readings.claim(v)
	if read == nil {
		return running, reason
	}
	//nolint:contextcheck // a vault is read for the life of the window, under behind, not under the count that asked
	go a.walk(behind, read, v, a.hasVault(ctx, v))
	return true, ""
}

// hasVault is whether the index already holds this vault.
func (a *API) hasVault(ctx context.Context, v domain.Vault) bool {
	if a.Notes == nil {
		return false
	}
	held, err := a.Notes.Holds(ctx, v.ID)
	return err == nil && held
}

// walk is one vault read, reported as work for as long as it runs. A reading
// that finished wakes the counts, which is what puts the vault's numbers on the
// page.
//
// held says the index already carries the vault. A vault it does not carry has
// nothing to show until this is done and the person is waiting on it, so that
// work is drawn the moment it begins; a vault it carries is work drawn once it
// has lasted.
func (a *API) walk(ctx context.Context, read ReadVault, v domain.Vault, held bool) {
	at := task.Task{
		ID:    "reading\t" + string(v.ID),
		Doing: "Reading the vault",
		About: v.Name,
		Asked: !held,
	}
	a.say(at)

	err := read(ctx, v)

	a.readings.finish(v, err)

	if err != nil {
		at.Error = err.Error()
		a.say(at)
		return
	}
	a.finishTask(at.ID)
	a.ReportChange()
}

// Forget lets go of why a vault could not be read. A vault that moved
// underneath the window is read again.
func (a *API) Forget(vaultID domain.VaultID) {
	a.readings.forget(vaultID)
}
