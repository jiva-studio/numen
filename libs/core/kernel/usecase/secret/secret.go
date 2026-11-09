package secret

import (
	"context"
	"errors"
	"fmt"
	"slices"

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

// operation is what every use case of this package is built from.
type operation struct {
	ledger  pluginport.SecretLedger
	keys    pluginport.KeyStore
	plugins pluginport.SecretPlugins
	gate    *Gate
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

func (o operation) apply(ctx context.Context, change func() (readers []string, err error)) error {
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

// GetEntryName returns the name the value of a secret is stored under in the key store, which the
// host reads it by when it sends the value to a plugin.
func GetEntryName(id string) string { return "secret/" + id }

func newInvalidError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

func newNotFoundError(id string) error { return fmt.Errorf("%w: %s", ErrNotFound, id) }

func (o operation) setValue(ctx context.Context, id, value string) error {
	if o.keys == nil {
		return fmt.Errorf("%w: %w", ErrKeyStore, pluginport.ErrNoKeyStore)
	}
	if err := o.keys.SetKey(ctx, GetEntryName(id), value); err != nil {
		return fmt.Errorf("%w: store the value of %s: %w", ErrKeyStore, id, err)
	}
	return nil
}

func (o operation) deleteValue(ctx context.Context, id string) error {
	if o.keys == nil {
		return nil
	}
	if err := o.keys.DeleteKey(ctx, GetEntryName(id)); err != nil && !errors.Is(err, pluginport.ErrNoKey) {
		return fmt.Errorf("%w: delete the value of %s: %w", ErrKeyStore, id, err)
	}
	return nil
}

func getIndex(list []pluginport.Secret, id string) int {
	return slices.IndexFunc(list, func(one pluginport.Secret) bool { return one.ID == id })
}

func isNameTaken(list []pluginport.Secret, group, name, exceptID string) bool {
	return slices.ContainsFunc(list, func(one pluginport.Secret) bool {
		return one.Group == group && one.Name == name && one.ID != exceptID
	})
}

func checkName(kind, name string) error {
	if !pluginport.IsValidSecretName(name) {
		return newInvalidError("the %s is empty, has leading or trailing space or is over %d characters", kind, pluginport.MaxSecretNameLength)
	}
	return nil
}

func listPluginIDs(allowed []pluginport.DeclarationKey) []string {
	var ids []string
	for _, one := range allowed {
		if !slices.Contains(ids, one.Plugin) {
			ids = append(ids, one.Plugin)
		}
	}
	return ids
}

// Secrets holds every use case on the list of secrets.
type Secrets struct {
	Create      *Create
	Rename      *Rename
	Write       *Write
	Delete      *Delete
	DeleteGroup *DeleteGroup
	Move        *Move
	RenameGroup *RenameGroup
	Allow       *Allow
	Deny        *Deny
	List        *List
	ListStates  *ListStates
	Prune       *Prune
}

// NewSecrets returns the use cases over the list, the key store and the plugin host. newID makes
// the id of a new secret and credentialsPath answers where the values are kept when it is a file.
// Use cases that share a gate never change the key store and the list at once; a nil gate is a new one.
func NewSecrets(
	ledger pluginport.SecretLedger, keys pluginport.KeyStore, plugins pluginport.SecretPlugins,
	newID func() (string, error), credentialsPath func() string, gate *Gate,
) *Secrets {
	if gate == nil {
		gate = NewGate()
	}
	base := operation{ledger: ledger, keys: keys, plugins: plugins, gate: gate}
	return &Secrets{
		Create:      &Create{operation: base, newID: newID},
		Rename:      &Rename{operation: base},
		Write:       &Write{operation: base},
		Delete:      &Delete{operation: base},
		DeleteGroup: &DeleteGroup{operation: base},
		Move:        &Move{operation: base},
		RenameGroup: &RenameGroup{operation: base},
		Allow:       &Allow{operation: base},
		Deny:        &Deny{operation: base},
		List:        &List{operation: base, credentialsPath: credentialsPath},
		ListStates:  &ListStates{operation: base},
		Prune:       &Prune{operation: base},
	}
}

// NoPlugins replaces the plugin host where the installation has none: no plugin declares a
// secret, and an operation only changes the list.
type NoPlugins struct{}

// ListDeclarations returns no declaration.
func (NoPlugins) ListDeclarations() []pluginport.DeclarationKey { return nil }

// Refresh configures no plugin.
func (NoPlugins) Refresh(context.Context, ...string) error { return nil }
