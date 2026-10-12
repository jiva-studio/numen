package flashcards

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// IndexVault is a vault walked into the index.
type IndexVault func(ctx context.Context, v domain.Vault) error

// readings is which vaults this window has read into the index: the ones read
// since it opened, the ones being read now, and why the last reading of one
// failed.
type readings struct {
	mu sync.Mutex

	// indexVault is how a vault is brought up to date in the index, and ctx is the
	// life those readings run for.
	indexVault IndexVault
	ctx        context.Context

	indexed  map[domain.VaultID]bool
	running  map[domain.VaultID]bool
	failures map[domain.VaultID]string
}

// setIndexer is how a vault is read from now on, and the life those readings run
// for.
func (r *readings) setIndexer(ctx context.Context, read IndexVault) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ctx, r.indexVault = ctx, read
}

// clear lets go of why a vault could not be read, and that it was read at all.
func (r *readings) clear(id domain.VaultID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.failures, id)
	delete(r.indexed, id)
}

// tryStart claims a vault for reading. It hands back the reader and the life to
// run it under where this call is the one that begins it, and no reader where
// the vault has been read, is being read, could not be read, or where this
// window reads nothing.
func (r *readings) tryStart(
	v domain.Vault,
) (read IndexVault, under context.Context, underway bool, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.indexed[v.ID] {
		return nil, nil, false, ""
	}
	if why, told := r.failures[v.ID]; told {
		return nil, nil, false, why
	}
	if r.running[v.ID] {
		return nil, nil, true, ""
	}
	if r.indexVault == nil {
		return nil, nil, false, ""
	}
	if r.running == nil {
		r.running = make(map[domain.VaultID]bool)
	}
	r.running[v.ID] = true
	return r.indexVault, r.ctx, true, ""
}

// record is what one reading came to: the vault is read, or it is why it is not.
func (r *readings) record(v domain.Vault, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.running, v.ID)
	if err != nil {
		if r.failures == nil {
			r.failures = make(map[domain.VaultID]string)
		}
		r.failures[v.ID] = err.Error()
		return
	}
	if r.indexed == nil {
		r.indexed = make(map[domain.VaultID]bool)
	}
	r.indexed[v.ID] = true
}

// Reading is how a vault is brought up to date in the index, and the life those
// readings run for. A window naming none counts a vault from the index as it
// stands.
func (a *API) Reading(ctx context.Context, read IndexVault) {
	a.readings.setIndexer(ctx, read)
}

// reading brings a vault up to date in the index. It says whether a reading of
// that vault is running now, and why the last one failed, when one did.
//
// A vault is read once for the life of the window, and the watcher carries it
// from there. One vault is read once at a time however many counts ask for it,
// and a reading that failed is not begun again until the vault moves.
func (a *API) reading(ctx context.Context, v domain.Vault) (underway bool, reason string) {
	read, behind, running, reason := a.readings.tryStart(v)
	if read == nil {
		return running, reason
	}
	//nolint:contextcheck // a vault is read for the life of the window, under behind, not under the count that asked
	go a.runIndexing(behind, read, v, a.hasVault(ctx, v))
	return true, ""
}

// hasVault is whether the index already holds this vault.
func (a *API) hasVault(ctx context.Context, v domain.Vault) bool {
	if a.Notes == nil {
		return false
	}
	held, err := a.Notes.Has(ctx, v.ID)
	return err == nil && held
}

// runIndexing is one vault read, reported as work for as long as it runs. A reading
// that finished wakes the counts, which is what puts the vault's numbers on the
// page.
//
// held says the index already carries the vault. A vault it does not carry has
// nothing to show until this is done and the person is waiting on it, so that
// work is drawn the moment it begins; a vault it carries is work drawn once it
// has lasted.
func (a *API) runIndexing(ctx context.Context, read IndexVault, v domain.Vault, held bool) {
	at := task.Task{
		ID:      "reading\t" + string(v.ID),
		Doing:   "Reading the vault",
		About:   v.Name,
		IsAsked: !held,
	}
	a.reportTask(at)

	err := read(ctx, v)

	a.readings.record(v, err)

	if err != nil {
		at.Error = err.Error()
		a.reportTask(at)
		return
	}
	a.finishTask(at.ID)
	a.ReportChange()
}

// ClearIndexFailure lets go of why a vault could not be read. A vault that moved
// underneath the window is read again.
func (a *API) ClearIndexFailure(vaultID domain.VaultID) {
	a.readings.clear(vaultID)
}
