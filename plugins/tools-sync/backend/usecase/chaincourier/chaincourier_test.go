package chaincourier_test

import (
	"bytes"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/chaincourier"
)

func newComputer(t *testing.T) (*synctest.Computer, []string) {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	return synctest.NewFirstComputer(t, server, clock, "Test", words), words
}

func parseWords(t *testing.T, words []string) identity.RecoveryKey {
	t.Helper()
	parsed, err := identity.ParseRecoveryWords(words)
	if err != nil {
		t.Fatal(err)
	}
	root, err := parsed.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPublish_DoesNothingWhenCoreHasNoPendingChain(t *testing.T) {
	t.Parallel()
	c, _ := newComputer(t)

	published, err := chaincourier.NewPublish(c.ChainDeps()).Execute(t.Context())

	if err != nil || published {
		t.Errorf("published = %v, error = %v, want nothing published", published, err)
	}
}

func TestAdopt_PassesLongerServerChainToCore(t *testing.T) {
	t.Parallel()
	c, words := newComputer(t)
	own, err := identity.ParseChain(c.Fake.Public().Chain)
	if err != nil {
		t.Fatal(err)
	}
	other, err := identity.NewDeviceKey(bytes.NewReader(bytes.Repeat([]byte{3}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	longer, err := own.AppendDevice(parseWords(t, words), other.Public(), "Phone", synctest.GetStart())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Cloud.AppendChain(t.Context(), longer.Bytes()); err != nil {
		t.Fatalf("AppendChain() = %v", err)
	}

	adopted, err := chaincourier.NewAdopt(c.ChainDeps()).Execute(t.Context())

	if err != nil || !adopted {
		t.Fatalf("adopted = %v, error = %v, want the longer chain adopted", adopted, err)
	}
	if got := c.Fake.Public().Chain; !bytes.Equal(got, longer.Bytes()) {
		t.Errorf("the core holds %d bytes of chain, want the chain of the server (%d)", len(got), len(longer.Bytes()))
	}
}

func TestAdopt_DoesNothingWhenTheChainsAreTheSameLength(t *testing.T) {
	t.Parallel()
	c, _ := newComputer(t)

	adopted, err := chaincourier.NewAdopt(c.ChainDeps()).Execute(t.Context())

	if err != nil || adopted {
		t.Errorf("adopted = %v, error = %v, want nothing adopted", adopted, err)
	}
}
