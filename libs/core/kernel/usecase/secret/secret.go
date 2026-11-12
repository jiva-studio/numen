package secret

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

var (
	// ErrInvalid is a request whose name, value or target is not acceptable.
	ErrInvalid = errors.New("invalid secret request")
	// ErrNotFound is a request for a secret that is not in the list.
	ErrNotFound = errors.New("no such secret")
	// ErrKeyStore is a key store that fails or that this machine does not have.
	ErrKeyStore = errors.New("the key store failed")
)

// Operation is what every use case of this package is built from: the list of secrets, the key store, the plugin
// host, and the gate that keeps their updates apart.
type Operation struct {
	ledger  pluginport.SecretLedger
	keys    pluginport.KeyStore
	plugins pluginport.SecretPlugins
	gate    *Gate
}

// NewOperation returns the ground of the use cases over the list, the key store and the plugin host. Use
// cases that share a gate never change the key store and the list at once; a nil gate is a new one.
func NewOperation(ledger pluginport.SecretLedger, keys pluginport.KeyStore, plugins pluginport.SecretPlugins, gate *Gate) Operation {
	if gate == nil {
		gate = NewGate()
	}
	return Operation{ledger: ledger, keys: keys, plugins: plugins, gate: gate}
}

// Gate serializes secret store updates, list writes, and plugin configuration refreshes.
type Gate struct{ slot chan struct{} }

// NewGate returns a gate that no operation holds.
func NewGate() *Gate { return &Gate{slot: make(chan struct{}, 1)} }

// Enter waits for the gate. It returns the context error, holding nothing, when ctx ends first. A
// nil gate is always free.
func (g *Gate) Enter(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if g == nil {
		return nil
	}
	select {
	case g.slot <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Leave frees a gate that Enter returned nil for.
func (g *Gate) Leave() {
	if g != nil {
		<-g.slot
	}
}

func (o Operation) apply(ctx context.Context, change func() (readers []string, err error)) error {
	if err := o.gate.Enter(ctx); err != nil {
		return err
	}
	defer o.gate.Leave()
	readers, err := change()
	if err != nil {
		return err
	}
	return o.plugins.Refresh(ctx, readers...)
}

func newInvalidError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

func newNotFoundError(id string) error { return fmt.Errorf("%w: %s", ErrNotFound, id) }

func (o Operation) setValue(ctx context.Context, id, value string) error {
	if o.keys == nil {
		return fmt.Errorf("%w: %w", ErrKeyStore, plugin.ErrNoKeyStore)
	}
	if err := o.keys.SetKey(ctx, plugin.GetSecretEntryName(id), value); err != nil {
		return fmt.Errorf("%w: store the value of %s: %w", ErrKeyStore, id, err)
	}
	return nil
}

func (o Operation) deleteValue(ctx context.Context, id string) error {
	if o.keys == nil {
		return nil
	}
	if err := o.keys.DeleteKey(ctx, plugin.GetSecretEntryName(id)); err != nil && !errors.Is(err, plugin.ErrNoKey) {
		return fmt.Errorf("%w: delete the value of %s: %w", ErrKeyStore, id, err)
	}
	return nil
}

func getIndex(list []plugin.Secret, id string) int {
	return slices.IndexFunc(list, func(one plugin.Secret) bool { return one.ID == id })
}

func isNameTaken(list []plugin.Secret, group, name, exceptID string) bool {
	return slices.ContainsFunc(list, func(one plugin.Secret) bool {
		return one.Group == group && one.Name == name && one.ID != exceptID
	})
}

func checkName(kind, name string) error {
	if !plugin.IsValidSecretName(name) {
		return newInvalidError("the %s is empty, has leading or trailing space or is over %d characters", kind, plugin.MaxSecretNameLength)
	}
	return nil
}

func listPluginIDs(allowed []plugin.DeclarationKey) []string {
	var ids []string
	for _, one := range allowed {
		if !slices.Contains(ids, one.Plugin) {
			ids = append(ids, one.Plugin)
		}
	}
	return ids
}
