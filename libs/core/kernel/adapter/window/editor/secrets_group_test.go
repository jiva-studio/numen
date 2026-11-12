package editor_test

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

func (k *secretKeys) readCount() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.reads
}

func TestGettingTheSettingsReadsNoValueFromTheKeyStore(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	w.create(t, "openai", "Work", "value-work")
	w.create(t, "openai", "Home", "value-home")
	before := w.keys.readCount()

	w.get(t)

	if after := w.keys.readCount(); after != before {
		t.Errorf("GetSettings read %d values from the key store, want none", after-before)
	}
}

func TestListingTheStatesTellsWhichSecretsHaveAValue(t *testing.T) {
	const imported = `{"secrets":[{"id":"s0000abcd","name":"Imported","group":"openrouter","allowed":[]}]}`
	w := openSecretsWindow(t, imported)
	one := w.create(t, "openai", "Work", "value-work")

	if !w.hasValue(t, one) || w.hasValue(t, "s0000abcd") {
		t.Error("the states do not match the key store")
	}
}

func TestDeletingAGroupRemovesItsSecretsAndConfiguresTheReadersOnce(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	one := w.create(t, "openai", "Shared", "value-shared")
	two := w.create(t, "openai", "Second", "value-second")
	other := w.create(t, "openrouter", "Kept", "value-kept")
	w.allow(t, one, betaID, "apiKey", "openai")
	w.allow(t, two, betaID, "apiKey", "openai")
	w.assertIDs(t, betaID, "apiKey", one, two)
	before := w.counts()

	if _, err := w.client.DeleteSecretGroup(t.Context(), connect.NewRequest(&v1.DeleteSecretGroupRequest{Group: "openai"})); err != nil {
		t.Fatal(err)
	}

	w.assertIDs(t, betaID, "apiKey")
	if got := ids(w.secrets(t)); len(got) != 1 || got[0] != other {
		t.Errorf("the list is %v, want only the secret of the other group", got)
	}
	if after := w.counts(); after[betaID] != before[betaID]+1 {
		t.Errorf("the reader was configured %d times, want once", after[betaID]-before[betaID])
	}
	if w.hasValue(t, other) != true || len(w.keys.snapshot()) != 1 {
		t.Errorf("the key store holds %v", w.keys.snapshot())
	}
}

func TestDeletingAGroupWithNoSecretIsNotFound(t *testing.T) {
	w := openSecretsWindow(t, `{}`)
	if _, err := w.client.DeleteSecretGroup(t.Context(), connect.NewRequest(&v1.DeleteSecretGroupRequest{Group: "nothing"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("error = %v, want NotFound", err)
	}
}
