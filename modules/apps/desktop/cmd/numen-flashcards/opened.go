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

// errGoing is work asked for once the window has begun closing.
var errGoing = errors.New("the window is closing")

// openVaults is every vault this window has open.
//
// A vault is opened once, for the life of the window: its watch is started and
// left running, its walk is asked for from here, and what this window writes
// into it is levelled through the same opening.
type openVaults struct {
	cfg container.Config
	db  *container.Index
	// ctx is the life a vault stays open for. It outlives the question that
	// first asked after the vault.
	ctx context.Context
	// record is called with the vault the index has just been brought level with.
	record func(domain.Vault)
	out    io.Writer

	// running is every walk, every watch and every levelling this window has
	// over a vault. They write to the index, so they are waited for before it
	// closes.
	running sync.WaitGroup

	mu       sync.Mutex
	going    bool
	openings map[domain.VaultID]*vaultOpening
}

// vaultOpening is one vault's opening, made once however many ask for it.
type vaultOpening struct {
	once    sync.Once
	opening *container.VaultOpener
	open    *container.OpenVault
}

// wait lets go of every vault and holds until nothing is still writing to the
// index.
func (o *openVaults) wait() {
	o.mu.Lock()
	o.going = true
	o.mu.Unlock()

	o.running.Wait()
}

// starts takes a piece of work on and says whether it may run. A window that is
// going takes none, so nothing begins writing after the index is waited for.
func (o *openVaults) starts() bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.going {
		return false
	}
	o.running.Add(1)
	return true
}

// of is the vault opened, and opens it the first time it is asked for. Opening
// one registers a watch over its whole tree, which is done outside the lock.
func (o *openVaults) of(v domain.Vault) *vaultOpening {
	o.mu.Lock()
	one, there := o.openings[v.ID]
	if !there {
		one = &vaultOpening{}
		if o.openings == nil {
			o.openings = map[domain.VaultID]*vaultOpening{}
		}
		o.openings[v.ID] = one
	}
	o.mu.Unlock()

	one.once.Do(func() { o.opens(v, one) })
	return one
}

// opens starts one vault's watch and leaves it running.
func (o *openVaults) opens(v domain.Vault, one *vaultOpening) {
	opening := o.cfg.VaultOpener(o.db)
	opening.Told = func(container.VaultChanges) { o.record(v) }
	opening.Trouble = func(err error) {
		if err != nil {
			fmt.Fprintf(o.out, "numen-flashcards: %s: %v\n", v.Name, err)
		}
	}

	open := opening.Begin(o.ctx, v)
	if why := open.Unwatched(); why != nil {
		fmt.Fprintf(o.out, "numen-flashcards: %s is not being followed: %v\n", v.Name, why)
	}
	one.opening, one.open = opening, open

	if o.starts() {
		go func() {
			defer o.running.Done()
			open.Run(o.ctx)
		}()
	}
}

// reads walks a vault into the index, saying how far it has got in the notes
// written.
func (o *openVaults) reads(ctx context.Context, v domain.Vault, progress func(int64)) error {
	if !o.starts() {
		return errGoing
	}
	defer o.running.Done()

	_, err := o.of(v).open.Read(ctx, func(indexed int) { progress(int64(indexed)) })
	return err
}

// level brings the paths a write touched up to date, through the opening of the
// vault they are in.
//
// A levelling writes to the index, so a window that is closing refuses one: the
// prose is on disk either way, and a write into a database being closed is
// worse than a search that has to be caught up on next time.
func (o *openVaults) level(ctx context.Context, v domain.Vault, paths []string) error {
	if !o.starts() {
		return errGoing
	}
	defer o.running.Done()

	return o.of(v).opening.Level(ctx, v, paths)
}
