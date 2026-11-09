package secret_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/secret"
)

// countingPlugins counts the rounds of configuration.
type countingPlugins struct {
	*plugins
	rounds int
}

func (p *countingPlugins) Refresh(ctx context.Context, forced ...string) error {
	p.rounds++
	return p.plugins.Refresh(ctx, forced...)
}

func newGroupFixture() fixture {
	return newFixture(
		pluginport.Secret{ID: "s00000001", Name: "One", Group: "g"},
		pluginport.Secret{ID: "s00000002", Name: "Other", Group: "h"},
		pluginport.Secret{ID: "s00000003", Name: "Two", Group: "g", Allowed: []pluginport.DeclarationKey{getAlphaKey()}},
	)
}

func TestDeleteGroupRemovesSecretsAndValuesInOneRound(t *testing.T) {
	f := newGroupFixture()
	f.keys["secret/s00000001"] = "invented-one"
	f.keys["secret/s00000002"] = "invented-other"
	f.keys["secret/s00000003"] = "invented-two"
	host := &countingPlugins{plugins: f.plugins}
	f.secrets = secret.NewSecrets(f.ledger, f.keys, host, nil, nil, nil)

	if err := f.secrets.DeleteGroup.Execute(t.Context(), "g"); err != nil {
		t.Fatal(err)
	}
	if got := idsOf(f.ledger.list); !slices.Equal(got, []string{"s00000002"}) {
		t.Errorf("list = %v", got)
	}
	if len(f.keys) != 1 || f.keys["secret/s00000002"] == "" {
		t.Errorf("store = %v, want only the value of the other group", f.keys)
	}
	if host.rounds != 1 {
		t.Errorf("rounds = %d, want 1", host.rounds)
	}
}

func TestDeleteGroupOfAGroupWithNoSecretIsNotFound(t *testing.T) {
	f := newGroupFixture()
	if err := f.secrets.DeleteGroup.Execute(t.Context(), "missing"); !errors.Is(err, secret.ErrNotFound) {
		t.Errorf("error = %v", err)
	}
}

func TestDeleteGroupWithFailingStoreRestoresEntries(t *testing.T) {
	f := newGroupFixture()
	f.keys["secret/s00000003"] = "invented-two"
	f.secrets = secret.NewSecrets(f.ledger, undeletableStore{f.keys}, f.plugins, nil, nil, nil)

	err := f.secrets.DeleteGroup.Execute(t.Context(), "g")
	if !errors.Is(err, secret.ErrKeyStore) || !errors.Is(err, errLocked) {
		t.Errorf("error = %v", err)
	}
	want := []pluginport.Secret{
		{ID: "s00000001", Name: "One", Group: "g"},
		{ID: "s00000002", Name: "Other", Group: "h"},
		{ID: "s00000003", Name: "Two", Group: "g", Allowed: []pluginport.DeclarationKey{getAlphaKey()}},
	}
	if !slices.EqualFunc(f.ledger.list, want, func(a, b pluginport.Secret) bool {
		return a.ID == b.ID && slices.Equal(a.Allowed, b.Allowed)
	}) {
		t.Errorf("list = %v, want %v", f.ledger.list, want)
	}
}
