package secret_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
)

// ledger is a list of secrets held in memory.
type ledger struct{ list []plugin.Secret }

func (l *ledger) GetSecrets() ([]plugin.Secret, error) { return slices.Clone(l.list), nil }

func (l *ledger) ChangeSecrets(change func([]plugin.Secret) ([]plugin.Secret, error)) error {
	updated, err := change(slices.Clone(l.list))
	if err != nil {
		return err
	}
	l.list = updated
	return nil
}

// plugins is a host that declares the given declarations and records the readers an operation names.
type plugins struct {
	mu       sync.Mutex
	declared []plugin.DeclarationKey
	readers  []string
}

func (p *plugins) ListDeclarations() []plugin.DeclarationKey { return p.declared }

func (p *plugins) Refresh(_ context.Context, forced ...string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.readers = forced
	return nil
}

// keys is a key store held in memory.
type keys map[string]string

func (k keys) GetKey(_ context.Context, name string) (string, error) {
	if value, found := k[name]; found {
		return value, nil
	}
	return "", plugin.ErrNoKey
}

func (k keys) SetKey(_ context.Context, name, value string) error { k[name] = value; return nil }

func (k keys) DeleteKey(_ context.Context, name string) error {
	if _, found := k[name]; !found {
		return plugin.ErrNoKey
	}
	delete(k, name)
	return nil
}

func getAlphaKey() plugin.DeclarationKey {
	return plugin.DeclarationKey{Plugin: "md.numen.alpha", Name: "apiKey", Group: "openrouter"}
}

func getBetaKey() plugin.DeclarationKey {
	return plugin.DeclarationKey{Plugin: "md.numen.beta", Name: "apiKey"}
}

type fixture struct {
	ledger  *ledger
	plugins *plugins
	keys    keys
	secrets *secrets
}

func newFixture(list ...plugin.Secret) fixture {
	f := fixture{ledger: &ledger{list: list}, plugins: &plugins{declared: []plugin.DeclarationKey{getAlphaKey(), getBetaKey()}}, keys: keys{}}
	counter := 0
	newID := func() (string, error) {
		counter++
		return "s0000000" + string(rune('0'+counter)), nil
	}
	f.secrets = newSecrets(f.ledger, f.keys, f.plugins, newID, func(context.Context) string { return "/credentials.json" }, nil)
	return f
}

func TestCreateAppendsSecretAndStoresValue(t *testing.T) {
	f := newFixture(plugin.Secret{ID: "s00000099", Name: "Old", Group: "openrouter"})

	id, err := f.secrets.Items.Create.Execute(t.Context(), "openrouter", "Work", "value-one")
	if err != nil {
		t.Fatal(err)
	}
	if got := idsOf(f.ledger.list); !slices.Equal(got, []string{"s00000099", id}) {
		t.Errorf("list = %v", got)
	}
	if f.keys["secret/"+id] != "value-one" {
		t.Errorf("store = %v", f.keys)
	}
	if len(f.ledger.list[1].Allowed) != 0 {
		t.Errorf("a new secret is allowed to %v", f.ledger.list[1].Allowed)
	}
}

func TestCreateRejectsInvalidRequestBeforeWrite(t *testing.T) {
	f := newFixture(plugin.Secret{ID: "s00000099", Name: "Old", Group: "openrouter"})
	for what, call := range map[string]func() error{
		"an empty group": func() error { _, err := f.secrets.Items.Create.Execute(t.Context(), "", "N", "v"); return err },
		"an empty name":  func() error { _, err := f.secrets.Items.Create.Execute(t.Context(), "g", "", "v"); return err },
		"a taken name": func() error {
			_, err := f.secrets.Items.Create.Execute(t.Context(), "openrouter", "Old", "v")
			return err
		},
		"an empty value": func() error { _, err := f.secrets.Items.Create.Execute(t.Context(), "g", "N", ""); return err },
		"a long name": func() error {
			_, err := f.secrets.Items.Create.Execute(t.Context(), "g", strings.Repeat("n", 65), "v")
			return err
		},
		"an untrimmed one": func() error { _, err := f.secrets.Items.Create.Execute(t.Context(), "g", "N ", "v"); return err },
	} {
		if err := call(); !errors.Is(err, secret.ErrInvalid) {
			t.Errorf("%s gave %v, want ErrInvalid", what, err)
		}
	}
	if len(f.keys) != 0 || len(f.ledger.list) != 1 {
		t.Errorf("a refused request changed the store %v or the list %v", f.keys, f.ledger.list)
	}
}

type failingStore struct{ keys }

var errLocked = errors.New("the keychain is locked")

func (failingStore) SetKey(context.Context, string, string) error { return errLocked }

func TestCreateWithFailingStoreRedactsValueAndSkipsWrite(t *testing.T) {
	f := newFixture()
	f.secrets = newSecrets(f.ledger, failingStore{keys{}}, f.plugins, func() (string, error) { return "s00000001", nil }, nil, nil)

	_, err := f.secrets.Items.Create.Execute(t.Context(), "g", "N", "value-that-must-stay-secret")
	if !errors.Is(err, secret.ErrKeyStore) || !errors.Is(err, errLocked) || strings.Contains(err.Error(), "value-that-must-stay-secret") {
		t.Errorf("error = %v", err)
	}
	if len(f.ledger.list) != 0 {
		t.Errorf("list = %v", f.ledger.list)
	}
}

func TestWriteNamesTheReadersOfTheSecretOnce(t *testing.T) {
	f := newFixture(plugin.Secret{ID: "s00000001", Name: "Work", Group: "openrouter", Allowed: []plugin.DeclarationKey{
		getAlphaKey(), {Plugin: "md.numen.alpha", Name: "other"}, getBetaKey(),
	}})
	if err := f.secrets.Items.Write.Execute(t.Context(), "s00000001", "value-new"); err != nil {
		t.Fatal(err)
	}
	if f.keys["secret/s00000001"] != "value-new" || !slices.Equal(f.plugins.readers, []string{"md.numen.alpha", "md.numen.beta"}) {
		t.Errorf("store %v, readers %v", f.keys, f.plugins.readers)
	}
	if err := f.secrets.Items.Write.Execute(t.Context(), "s00000001", ""); !errors.Is(err, secret.ErrInvalid) {
		t.Errorf("an empty value gave %v", err)
	}
	if err := f.secrets.Items.Write.Execute(t.Context(), "s00000042", "v"); !errors.Is(err, secret.ErrNotFound) {
		t.Errorf("an unknown id gave %v", err)
	}
}

func TestRenameKeepsTheIdPermissionsAndValue(t *testing.T) {
	f := newFixture(
		plugin.Secret{ID: "s00000001", Name: "Work", Group: "g", Allowed: []plugin.DeclarationKey{getAlphaKey()}},
		plugin.Secret{ID: "s00000002", Name: "Home", Group: "g"},
	)
	f.keys["secret/s00000001"] = "v"
	if err := f.secrets.Items.Rename.Execute(t.Context(), "s00000001", "Office"); err != nil {
		t.Fatal(err)
	}
	if got := f.ledger.list[0]; got.Name != "Office" || len(got.Allowed) != 1 || f.keys["secret/s00000001"] != "v" {
		t.Errorf("secret = %v, store %v", got, f.keys)
	}
	if err := f.secrets.Items.Rename.Execute(t.Context(), "s00000001", "Home"); !errors.Is(err, secret.ErrInvalid) {
		t.Errorf("a taken name gave %v", err)
	}
}

func TestDeleteRemovesEntryAndToleratesMissingValue(t *testing.T) {
	f := newFixture(plugin.Secret{ID: "s00000001", Name: "Work", Group: "g"})
	if err := f.secrets.Items.Delete.Execute(t.Context(), "s00000001"); err != nil {
		t.Fatalf("a delete with no value in the store: %v", err)
	}
	if len(f.ledger.list) != 0 {
		t.Errorf("list = %v", f.ledger.list)
	}
	if err := f.secrets.Items.Delete.Execute(t.Context(), "s00000001"); !errors.Is(err, secret.ErrNotFound) {
		t.Errorf("a second delete gave %v", err)
	}
}

func TestMoveStaysInsideTheGroup(t *testing.T) {
	f := newFixture(
		plugin.Secret{ID: "s00000001", Group: "a"}, plugin.Secret{ID: "s00000002", Group: "b"},
		plugin.Secret{ID: "s00000003", Group: "a"}, plugin.Secret{ID: "s00000004", Group: "a"},
	)
	if err := f.secrets.Items.Move.Execute(t.Context(), "s00000004", "s00000001"); err != nil {
		t.Fatal(err)
	}
	if got := idsOf(f.ledger.list); !slices.Equal(got, []string{"s00000004", "s00000001", "s00000002", "s00000003"}) {
		t.Errorf("list = %v", got)
	}
	if err := f.secrets.Items.Move.Execute(t.Context(), "s00000004", ""); err != nil {
		t.Fatal(err)
	}
	if got := idsOf(f.ledger.list); got[len(got)-1] != "s00000004" {
		t.Errorf("list = %v, want the moved one last", got)
	}
	if err := f.secrets.Items.Move.Execute(t.Context(), "s00000004", "s00000002"); !errors.Is(err, secret.ErrInvalid) {
		t.Errorf("a move into another group gave %v", err)
	}
	if err := f.secrets.Items.Move.Execute(t.Context(), "s00000004", "s00000077"); !errors.Is(err, secret.ErrNotFound) {
		t.Errorf("a move before an unknown secret gave %v", err)
	}
}

func TestRenameGroupUpdatesSecretsAndValidatesTargetName(t *testing.T) {
	f := newFixture(
		plugin.Secret{ID: "s00000001", Name: "A", Group: "work"}, plugin.Secret{ID: "s00000002", Name: "B", Group: "work"},
		plugin.Secret{ID: "s00000003", Name: "C", Group: "home"},
	)
	if err := f.secrets.Groups.Rename.Execute(t.Context(), "work", "office"); err != nil {
		t.Fatal(err)
	}
	if f.ledger.list[0].Group != "office" || f.ledger.list[1].Group != "office" || f.ledger.list[2].Group != "home" {
		t.Errorf("list = %v", f.ledger.list)
	}
	for what, call := range map[string]func() error{
		"a taken name":         func() error { return f.secrets.Groups.Rename.Execute(t.Context(), "office", "home") },
		"an asked-for name":    func() error { return f.secrets.Groups.Rename.Execute(t.Context(), "office", "openrouter") },
		"an asked-for group":   func() error { return f.secrets.Groups.Rename.Execute(t.Context(), "openrouter", "mine") },
		"an empty name":        func() error { return f.secrets.Groups.Rename.Execute(t.Context(), "office", "") },
		"a group of no secret": func() error { return f.secrets.Groups.Rename.Execute(t.Context(), "absent", "mine") },
	} {
		if err := call(); err == nil {
			t.Errorf("%s was accepted", what)
		}
	}
}

func TestAllowAndDenyAreIdempotentAndAllowChecksTheDeclaration(t *testing.T) {
	f := newFixture(plugin.Secret{ID: "s00000001", Name: "Work", Group: "openrouter"})
	for range 2 {
		if err := f.secrets.Access.Allow.Execute(t.Context(), "s00000001", getAlphaKey()); err != nil {
			t.Fatal(err)
		}
	}
	if got := f.ledger.list[0].Allowed; len(got) != 1 || got[0] != getAlphaKey() {
		t.Errorf("allowed = %v", got)
	}
	absent := plugin.DeclarationKey{Plugin: "md.numen.alpha", Name: "apiKey", Group: "mistral"}
	if err := f.secrets.Access.Allow.Execute(t.Context(), "s00000001", absent); !errors.Is(err, secret.ErrInvalid) {
		t.Errorf("an undeclared permission gave %v", err)
	}
	if err := f.secrets.Access.Allow.Execute(t.Context(), "s00000042", getAlphaKey()); !errors.Is(err, secret.ErrNotFound) {
		t.Errorf("an unknown secret gave %v", err)
	}
	for range 2 {
		if err := f.secrets.Access.Deny.Execute(t.Context(), "s00000001", getAlphaKey()); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.ledger.list[0].Allowed) != 0 {
		t.Errorf("allowed = %v", f.ledger.list[0].Allowed)
	}
}

func TestPruneKeepsTheDeclaredPermissionsAndDropsTheRest(t *testing.T) {
	removed := plugin.DeclarationKey{Plugin: "md.numen.removed", Name: "apiKey"}
	f := newFixture(plugin.Secret{ID: "s00000001", Name: "Work", Group: "g", Allowed: []plugin.DeclarationKey{getAlphaKey(), removed}})
	if err := f.secrets.Prune.Execute([]plugin.DeclarationKey{getAlphaKey()}); err != nil {
		t.Fatal(err)
	}
	if got := f.ledger.list[0].Allowed; len(got) != 1 || got[0] != getAlphaKey() {
		t.Errorf("allowed = %v", got)
	}
}

type undeletableStore struct{ keys }

func (undeletableStore) DeleteKey(context.Context, string) error { return errLocked }

func TestDeleteWithFailingStoreRestoresEntryAndSkipsRefresh(t *testing.T) {
	list := []plugin.Secret{
		{ID: "s00000001", Name: "First", Group: "g"},
		{ID: "s00000002", Name: "Work", Group: "g", Allowed: []plugin.DeclarationKey{getAlphaKey()}},
		{ID: "s00000003", Name: "Last", Group: "g"},
	}
	f := newFixture(list...)
	f.keys["secret/s00000002"] = "invented-value"
	f.secrets = newSecrets(f.ledger, undeletableStore{f.keys}, f.plugins, nil, nil, nil)

	err := f.secrets.Items.Delete.Execute(t.Context(), "s00000002")
	if !errors.Is(err, secret.ErrKeyStore) || !errors.Is(err, errLocked) {
		t.Errorf("error = %v", err)
	}
	if !slices.EqualFunc(f.ledger.list, list, func(a, b plugin.Secret) bool {
		return a.ID == b.ID && slices.Equal(a.Allowed, b.Allowed)
	}) {
		t.Errorf("list = %v, want %v", f.ledger.list, list)
	}
	if len(f.plugins.readers) != 0 {
		t.Errorf("readers = %v, want nobody", f.plugins.readers)
	}
}
