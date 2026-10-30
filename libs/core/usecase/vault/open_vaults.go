package vault

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

// OpenerFactory provides a VaultOpener for an index.
type OpenerFactory[I any] interface {
	VaultOpener(I) *VaultOpener
}

// OpenVaults is every vault this window has open.
type OpenVaults[C OpenerFactory[I], I any] struct {
	Config      C
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

// CloseAndWait lets go of every vault and holds until nothing is still writing to the index.
func (o *OpenVaults[C, I]) CloseAndWait() {
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

func (o *OpenVaults[C, I]) tryBeginWork() bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.isClosing {
		return false
	}
	o.running.Add(1)
	return true
}

func (o *OpenVaults[C, I]) getOpenedVault(ctx context.Context, v domain.Vault) *openedVault {
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

func (o *OpenVaults[C, I]) openVault(ctx context.Context, v domain.Vault, one *openedVault) {
	opening := o.Config.VaultOpener(o.Index)
	opening.Told = func(VaultChanges) { o.RecordVault(v) }
	opening.ErrorHandler = func(err error) {
		if err != nil {
			o.writeOutput("%s: %s: %v\n", o.Label, v.Name, err)
		}
	}

	lifetime, cancel := context.WithCancel(context.WithoutCancel(ctx))
	o.mu.Lock()
	if o.isClosing {
		cancel()
	} else {
		o.cancels = append(o.cancels, cancel)
	}
	o.mu.Unlock()

	open := opening.Begin(lifetime, v)
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

// writeOutput writes a line to Out. The error of a failed write is kept, and the next
// IndexVault or UpdateIndex returns it.
func (o *OpenVaults[C, I]) writeOutput(format string, args ...any) {
	if _, err := fmt.Fprintf(o.Out, format, args...); err != nil {
		o.mu.Lock()
		o.outputErr = errors.Join(o.outputErr, err)
		o.mu.Unlock()
	}
}

// takeOutputError returns the errors of failed writes to Out and forgets them.
func (o *OpenVaults[C, I]) takeOutputError() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	err := o.outputErr
	o.outputErr = nil
	return err
}

// IndexVault walks a vault into the index.
func (o *OpenVaults[C, I]) IndexVault(ctx context.Context, v domain.Vault) error {
	if !o.tryBeginWork() {
		return ErrClosing
	}
	defer o.running.Done()

	_, err := o.getOpenedVault(ctx, v).open.Read(ctx, nil)
	return errors.Join(err, o.takeOutputError())
}

// UpdateIndex brings the paths a write touched up to date, through the opening of the vault they are in.
func (o *OpenVaults[C, I]) UpdateIndex(ctx context.Context, v domain.Vault, paths []string) error {
	if !o.tryBeginWork() {
		return ErrClosing
	}
	defer o.running.Done()

	err := o.getOpenedVault(ctx, v).opening.Level(ctx, v, paths)
	return errors.Join(err, o.takeOutputError())
}

// OpenedCount returns the number of opened vaults.
func (o *OpenVaults[C, I]) OpenedCount() int { return len(o.opened) }
