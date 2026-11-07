package secret_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/secret"
)

// The listing reports whether the installation has a key store.
func TestListReportsWhetherTheInstallationHasAKeyStore(t *testing.T) {
	f := newFixture()
	stored, err := f.secrets.List.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !stored.HasKeyStore {
		t.Error("HasKeyStore is false for an installation stored a key store")
	}

	noStore := secret.NewSecrets(f.ledger, nil, f.plugins, nil, nil)
	listing, err := noStore.List.Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if listing.HasKeyStore {
		t.Error("HasKeyStore is true for an installation stored no key store")
	}
}
