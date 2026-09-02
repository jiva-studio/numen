package flashcardsui

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// Read is a vault walked into the index. It is handed how far it has got as it
// goes, counted in the notes written.
type Read func(ctx context.Context, v domain.Vault, got func(notes int64)) error

// Reading is how a vault is brought up to date in the index, and the life those
// readings run for. A window naming none counts a vault from the index as it
// stands.
func (a *API) Reading(ctx context.Context, read Read) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.behind, a.reads = ctx, read
}

// reading brings a vault up to date in the index. It says whether a reading of
// that vault is running now, and why the last one failed, when one did.
//
// A vault is read once for the life of the window, and the watcher carries it
// from there. One vault is read once at a time however many counts ask for it,
// and a reading that failed is not begun again until the vault moves.
func (a *API) reading(ctx context.Context, v domain.Vault) (underway bool, failed string) {
	a.mu.Lock()
	if a.walked[v.ID] {
		a.mu.Unlock()
		return false, ""
	}
	if why, told := a.unreadable[v.ID]; told {
		a.mu.Unlock()
		return false, why
	}
	if a.underway[v.ID] {
		a.mu.Unlock()
		return true, ""
	}
	read, behind := a.reads, a.behind
	if read == nil {
		a.mu.Unlock()
		return false, ""
	}
	if a.underway == nil {
		a.underway = make(map[string]bool)
	}
	a.underway[v.ID] = true
	a.mu.Unlock()

	go a.walk(behind, read, v, a.carries(ctx, v))
	return true, ""
}

// carries is whether the index already holds this vault.
func (a *API) carries(ctx context.Context, v domain.Vault) bool {
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
func (a *API) walk(ctx context.Context, read Read, v domain.Vault, held bool) {
	at := task.Task{
		ID:    "reading\t" + v.ID,
		Doing: "Reading the vault",
		About: v.Name,
		Asked: !held,
	}
	a.say(at)

	err := read(ctx, v, func(notes int64) {
		at.Done = notes
		a.say(at)
	})

	a.mu.Lock()
	delete(a.underway, v.ID)
	if err != nil {
		if a.unreadable == nil {
			a.unreadable = make(map[string]string)
		}
		a.unreadable[v.ID] = err.Error()
	} else {
		if a.walked == nil {
			a.walked = make(map[string]bool)
		}
		a.walked[v.ID] = true
	}
	a.mu.Unlock()

	if err != nil {
		at.Failed = err.Error()
		a.say(at)
		return
	}
	a.finished(at.ID)
	a.Moved()
}

// Forget lets go of why a vault could not be read. A vault that moved
// underneath the window is read again.
func (a *API) Forget(vaultID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.unreadable, vaultID)
}
