package flashcardsui

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// Read is a vault walked into the index. It is handed how far it has got as it
// goes, counted in the notes written.
type Read func(ctx context.Context, v domain.Vault, got func(notes int64)) error

// Reading is how a vault the index does not carry is brought into it, and the
// life those readings run for. A window naming none leaves such a vault
// uncounted.
func (a *API) Reading(ctx context.Context, read Read) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.behind, a.reads = ctx, read
}

// reading starts a vault into the index. It says whether a reading of that
// vault is now running and why the last one failed, when one did.
//
// One vault is read once at a time, however many counts ask for it, and a
// reading that failed is not begun again until the vault moves.
func (a *API) reading(v domain.Vault) (underway bool, failed string) {
	a.mu.Lock()
	if why, told := a.unreadable[v.ID]; told {
		a.mu.Unlock()
		return false, why
	}
	read, behind := a.reads, a.behind
	if read == nil {
		a.mu.Unlock()
		return false, ""
	}
	if a.underway[v.ID] {
		a.mu.Unlock()
		return true, ""
	}
	if a.underway == nil {
		a.underway = make(map[string]bool)
	}
	a.underway[v.ID] = true
	a.mu.Unlock()

	go a.walk(behind, read, v)
	return true, ""
}

// walk is one vault read, reported as work for as long as it runs. A reading
// that finished wakes the counts, which is what puts the vault's numbers on the
// page.
func (a *API) walk(ctx context.Context, read Read, v domain.Vault) {
	// Asked, because opening the window on this vault is what began the reading
	// and the person is waiting on it.
	at := task.Task{
		ID:    "reading\t" + v.ID,
		Doing: "Reading the vault",
		About: v.Name,
		Asked: true,
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

// forgetting lets go of why a vault could not be read. A vault that moved is
// read again.
func (a *API) forgetting() {
	a.mu.Lock()
	defer a.mu.Unlock()
	clear(a.unreadable)
}
