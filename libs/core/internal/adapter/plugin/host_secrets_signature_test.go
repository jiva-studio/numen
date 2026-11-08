package plugin

import (
	"context"
	"sync/atomic"
	"testing"
)

// countedKeys counts the reads of the key store it wraps.
type countedKeys struct {
	*keys
	reads atomic.Int32
}

func (c *countedKeys) GetKey(ctx context.Context, name string) (string, error) {
	c.reads.Add(1)
	return c.keys.GetKey(ctx, name)
}

// Telling whether a list of secrets changed asks the list and never the key store, which is a call
// to the system keychain for every secret.
func TestCheckingTheListOfSecretsDoesNotReadTheKeyStore(t *testing.T) {
	store := &countedKeys{keys: newKeys(map[string]string{"secret/" + secretID: fromKeychain})}
	host, _ := newSecretHost(t, store)
	host.SetSecretSource(allowOne(secretPluginID))
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	readsBefore := store.reads.Load()
	if readsBefore == 0 {
		t.Fatal("the Configure call carried no value")
	}

	if host.secrets.hasSecretsChanged(t.Context(), secretPluginID) {
		t.Error("an unchanged list is reported as changed")
	}
	if err := host.RefreshSecrets(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := store.reads.Load(); got != readsBefore {
		t.Errorf("the check read the key store %d times", got-readsBefore)
	}
}
