package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// errClosing is work asked for once the window has begun closing.
var errClosing = errors.New("the window is closing")

// openVaults is every vault this window has open.
//
// A vault is opened once, for the life of the window: its watch is started and
// left running, its walk is asked for from here, and what this window writes
// into it is levelled through the same opening.
type openVaults struct {
	cfg container.Config
	db  *container.Index
	// lifetime is the life a vault stays open for. It outlives the question that
	// first asked after the vault.
	lifetime context.Context
	// recordVault is called with the vault the index has just been brought
	// level with.
	recordVault func(domain.Vault)
	out         io.Writer

	// running is every walk, every watch and every levelling this window has
	// over a vault. They write to the index, so they are waited for before it
	// closes.
	running sync.WaitGroup

	mu        sync.Mutex
	isClosing bool
	opened    map[domain.VaultID]*openedVault
}

// openedVault is one vault's opening, made once however many ask for it.
type openedVault struct {
	once    sync.Once
	opening *container.VaultOpener
	open    *container.OpenVault
}

// closeAndWait lets go of every vault and holds until nothing is still writing to the
// index.
func (o *openVaults) closeAndWait() {
	o.mu.Lock()
	o.isClosing = true
	o.mu.Unlock()

	o.running.Wait()
}

// tryBeginWork takes a piece of work on and says whether it may run. A window that
// is going takes none, so nothing begins writing after the index is waited for.
func (o *openVaults) tryBeginWork() bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.isClosing {
		return false
	}
	o.running.Add(1)
	return true
}

// getOpenedVault is the vault opened, and opens it the first time it is asked for.
// Opening one registers a watch over its whole tree, which is done outside the
// lock.
func (o *openVaults) getOpenedVault(v domain.Vault) *openedVault {
	o.mu.Lock()
	one, there := o.opened[v.ID]
	if !there {
		one = &openedVault{}
		if o.opened == nil {
			o.opened = map[domain.VaultID]*openedVault{}
		}
		o.opened[v.ID] = one
	}
	o.mu.Unlock()

	one.once.Do(func() { o.openVault(v, one) })
	return one
}

// openVault starts one vault's watch and leaves it running.
func (o *openVaults) openVault(v domain.Vault, one *openedVault) {
	opening := o.cfg.VaultOpener(o.db)
	opening.Told = func(container.VaultChanges) { o.recordVault(v) }
	opening.ErrorHandler = func(err error) {
		if err != nil {
			fmt.Fprintf(o.out, "numen-flashcards: %s: %v\n", v.Name, err)
		}
	}

	open := opening.Begin(o.lifetime, v)
	if why := open.GetUnwatchedReason(); why != nil {
		fmt.Fprintf(o.out, "numen-flashcards: %s is not being followed: %v\n", v.Name, why)
	}
	one.opening, one.open = opening, open

	if o.tryBeginWork() {
		go func() {
			defer o.running.Done()
			open.Run(o.lifetime)
		}()
	}
}

// readVault walks a vault into the index.
func (o *openVaults) readVault(ctx context.Context, v domain.Vault) error {
	if !o.tryBeginWork() {
		return errClosing
	}
	defer o.running.Done()

	_, err := o.getOpenedVault(v).open.Read(ctx, nil)
	return err
}

// updateIndex brings the paths a write touched up to date, through the opening of the
// vault they are in.
//
// A levelling writes to the index, so a window that is closing refuses one: the
// prose is on disk either way, and a write into a database being closed is
// worse than a search that has to be caught up on next time.
func (o *openVaults) updateIndex(ctx context.Context, v domain.Vault, paths []string) error {
	if !o.tryBeginWork() {
		return errClosing
	}
	defer o.running.Done()

	return o.getOpenedVault(v).opening.Level(ctx, v, paths)
}
