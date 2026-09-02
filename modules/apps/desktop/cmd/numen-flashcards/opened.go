package main

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// opened is every vault this window has open.
//
// A vault is opened once, for the life of the window: its watch is started and
// left running, its walk is asked for from here, and what this window writes
// into it is levelled through the same opening.
type opened struct {
	cfg container.Config
	db  *container.Index
	// under is the life a vault stays open for. It outlives the question that
	// first asked after the vault.
	under context.Context
	// moved is called when the index and a vault are level again.
	moved func()
	out   io.Writer

	// running is every walk and every watch this window has over a vault. They
	// write to the index, so they are waited for before it closes.
	running sync.WaitGroup

	mu      sync.Mutex
	opening map[string]*container.Opening
	open    map[string]*container.Open
}

// wait holds until nothing this window opened is still writing.
func (o *opened) wait() { o.running.Wait() }

// of is the vault opened, and opens it the first time it is asked for.
func (o *opened) of(v domain.Vault) (*container.Opening, *container.Open) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if open, held := o.open[v.ID]; held {
		return o.opening[v.ID], open
	}

	opening := o.cfg.Opening(o.db)
	opening.Told = func(vault.Moved) { o.moved() }
	opening.Trouble = func(err error) {
		if err != nil {
			fmt.Fprintf(o.out, "numen-flashcards: %s: %v\n", v.Name, err)
		}
	}

	open, err := opening.Begin(o.under, v)
	if err != nil {
		fmt.Fprintf(o.out, "numen-flashcards: %s is not being followed: %v\n", v.Name, err)
	}
	// A window that is going follows nothing new: what is already running is
	// what the index waits for before it closes.
	if o.under.Err() == nil {
		o.running.Add(1)
		go func() {
			defer o.running.Done()
			open.Run(o.under)
		}()
	}

	if o.open == nil {
		o.opening, o.open = map[string]*container.Opening{}, map[string]*container.Open{}
	}
	o.opening[v.ID], o.open[v.ID] = opening, open
	return opening, open
}

// reads walks a vault into the index, saying how far it has got in the notes
// written. A window that is going walks nothing.
func (o *opened) reads(ctx context.Context, v domain.Vault, got func(int64)) error {
	if err := o.under.Err(); err != nil {
		return err
	}
	o.running.Add(1)
	defer o.running.Done()

	_, open := o.of(v)
	_, err := open.Read(ctx, func(res vault.ScanResult) { got(int64(res.Indexed)) })
	return err
}

// level brings the paths a write touched up to date, through the opening of the
// vault they are in.
func (o *opened) level(ctx context.Context, v domain.Vault, paths []string) error {
	opening, _ := o.of(v)
	return opening.Level(ctx, v, paths)
}
