package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrClosing is work asked for once the window has begun closing.
var ErrClosing = errors.New("the window is closing")

// OpenVaults is every vault this window has open.
//
// A vault is opened once, for the life of the window: its watch is started and
// left running, its walk is asked for from here, and what this window writes
// into it is levelled through the same opening.
type OpenVaults struct {
	Config Config
	Index  *Index
	// Label prefixes what is said about a vault.
	Label string
	// Lifetime is the life a vault stays open for. It outlives the question that
	// first asked after the vault.
	Lifetime context.Context
	// RecordVault is called with the vault the index has just been brought
	// level with.
	RecordVault func(domain.Vault)
	Out         io.Writer

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
	opening *VaultOpener
	open    *OpenVault
}

// CloseAndWait lets go of every vault and holds until nothing is still writing to the
// index.
func (o *OpenVaults) CloseAndWait() {
	o.mu.Lock()
	o.isClosing = true
	o.mu.Unlock()

	o.running.Wait()
}

// tryBeginWork takes a piece of work on and says whether it may run. A window that
// is going takes none, so nothing begins writing after the index is waited for.
func (o *OpenVaults) tryBeginWork() bool {
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
func (o *OpenVaults) getOpenedVault(v domain.Vault) *openedVault {
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
func (o *OpenVaults) openVault(v domain.Vault, one *openedVault) {
	opening := o.Config.VaultOpener(o.Index)
	opening.Told = func(VaultChanges) { o.RecordVault(v) }
	opening.ErrorHandler = func(err error) {
		if err != nil {
			fmt.Fprintf(o.Out, "%s: %s: %v\n", o.Label, v.Name, err)
		}
	}

	open := opening.Begin(o.Lifetime, v)
	if why := open.GetUnwatchedReason(); why != nil {
		fmt.Fprintf(o.Out, "%s: %s is not being followed: %v\n", o.Label, v.Name, why)
	}
	one.opening, one.open = opening, open

	if o.tryBeginWork() {
		go func() {
			defer o.running.Done()
			open.Run(o.Lifetime)
		}()
	}
}

// IndexVault walks a vault into the index.
func (o *OpenVaults) IndexVault(ctx context.Context, v domain.Vault) error {
	if !o.tryBeginWork() {
		return ErrClosing
	}
	defer o.running.Done()

	_, err := o.getOpenedVault(v).open.Read(ctx, nil)
	return err
}

// UpdateIndex brings the paths a write touched up to date, through the opening of the
// vault they are in.
//
// A levelling writes to the index, so a window that is closing refuses one: the
// prose is on disk either way, and a write into a database being closed is
// worse than a search that has to be caught up on next time.
func (o *OpenVaults) UpdateIndex(ctx context.Context, v domain.Vault, paths []string) error {
	if !o.tryBeginWork() {
		return ErrClosing
	}
	defer o.running.Done()

	return o.getOpenedVault(v).opening.Level(ctx, v, paths)
}
