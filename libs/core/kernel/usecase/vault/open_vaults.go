package vault

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// ErrClosing is returned for work requested after the window began closing.
var ErrClosing = errors.New("the window is closing")

// OpenVaults is every vault this window has open.
type OpenVaults[I any] struct {
	Opener      func(I) *VaultOpener
	Index       I
	Label       string
	RecordVault func(domain.Vault)
	Out         io.Writer

	running   sync.WaitGroup
	mu        sync.Mutex
	isClosing bool
	cancels   []context.CancelFunc
	opened    map[domain.VaultID]*openedVault
	outputErr error
}

type openedVault struct {
	once    sync.Once
	opening *VaultOpener
	open    *OpenVault
}

// CloseAndWait releases every vault and holds until nothing is still writing to the index.
func (o *OpenVaults[I]) CloseAndWait() {
	o.mu.Lock()
	o.isClosing = true
	cancels := o.cancels
	o.cancels = nil
	o.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	o.running.Wait()
}

func (o *OpenVaults[I]) tryBeginWork() bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.isClosing {
		return false
	}
	o.running.Add(1)
	return true
}

func (o *OpenVaults[I]) getOpenedVault(ctx context.Context, v domain.Vault) *openedVault {
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

	one.once.Do(func() { o.openVault(ctx, v, one) })
	return one
}

func (o *OpenVaults[I]) openVault(ctx context.Context, v domain.Vault, one *openedVault) {
	opening := o.Opener(o.Index)

	lifetime, cancel := context.WithCancel(context.WithoutCancel(ctx))
	o.mu.Lock()
	if o.isClosing {
		cancel()
	} else {
		o.cancels = append(o.cancels, cancel)
	}
	o.mu.Unlock()

	open := opening.Begin(lifetime, v, func(VaultChanges) { o.RecordVault(v) }, func(err error) {
		if err != nil {
			o.writeOutput("%s: %s: %v\n", o.Label, v.Name, err)
		}
	})
	if why := open.GetUnwatchedReason(); why != nil {
		o.writeOutput("%s: %s is not being followed: %v\n", o.Label, v.Name, why)
	}
	one.opening, one.open = opening, open

	if o.tryBeginWork() {
		go func() {
			defer o.running.Done()
			open.Run(lifetime)
		}()
	}
}

func (o *OpenVaults[I]) writeOutput(format string, args ...any) {
	if _, err := fmt.Fprintf(o.Out, format, args...); err != nil {
		o.mu.Lock()
		o.outputErr = errors.Join(o.outputErr, err)
		o.mu.Unlock()
	}
}

func (o *OpenVaults[I]) takeOutputError() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	err := o.outputErr
	o.outputErr = nil
	return err
}

// IndexVault scans a vault and adds its files to the index.
func (o *OpenVaults[I]) IndexVault(ctx context.Context, v domain.Vault) error {
	if !o.tryBeginWork() {
		return ErrClosing
	}
	defer o.running.Done()

	_, err := o.getOpenedVault(ctx, v).open.Read(ctx, nil)
	return errors.Join(err, o.takeOutputError())
}

// UpdateIndex brings the paths a write touched up to date, through the opening of the vault they are in.
func (o *OpenVaults[I]) UpdateIndex(ctx context.Context, v domain.Vault, paths []string) error {
	if !o.tryBeginWork() {
		return ErrClosing
	}
	defer o.running.Done()

	err := o.getOpenedVault(ctx, v).opening.UpdateIndex(ctx, v, paths)
	return errors.Join(err, o.takeOutputError())
}

// OpenedCount returns the number of opened vaults.
func (o *OpenVaults[I]) OpenedCount() int { return len(o.opened) }
