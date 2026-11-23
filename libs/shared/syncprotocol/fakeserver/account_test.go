package fakeserver_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
)

func TestGetAccountChain_ReturnsTheChainTheDeviceSignedInWith(t *testing.T) {
	t.Parallel()
	e := newEnv(t)

	chain, err := e.accounts.GetAccountChain(t.Context())

	if err != nil || !bytes.Equal(chain, e.account.chain.Bytes()) {
		t.Errorf("chain = %x, error = %v, want the chain of the sign-in", chain, err)
	}
}

func TestGetAccountChain_ReturnsTheLongerChainAfterAnotherDeviceSignedInWithIt(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	account := newOwnedAccount(t)
	first := account.addDevice(t, "First")
	one := newEnvOn(t, address, &account, first)
	shortChain := bytes.Clone(account.chain.Bytes())
	second := account.addDevice(t, "Second")
	newEnvOn(t, address, &account, second)

	chain, err := one.accounts.GetAccountChain(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(chain, shortChain) || !bytes.Equal(chain, account.chain.Bytes()) {
		t.Errorf("the chain has %d bytes, want the chain with both devices of %d bytes", len(chain), len(account.chain.Bytes()))
	}
}

func TestGetAccountChain_KeepsTheLongerChainWhenAnOlderDeviceSignsInAgainWithTheShorterOne(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	account := newOwnedAccount(t)
	first := account.addDevice(t, "First")
	shortChain := account.chain
	one := newEnvOn(t, address, &account, first)
	second := account.addDevice(t, "Second")
	newEnvOn(t, address, &account, second)
	_ = signIn(t, address, first, shortChain)

	chain, err := one.accounts.GetAccountChain(t.Context())

	if err != nil || !bytes.Equal(chain, account.chain.Bytes()) {
		t.Errorf("the chain has %d bytes, error = %v, want the longer chain of %d bytes", len(chain), err, len(account.chain.Bytes()))
	}
}

func TestGetAccountChain_WithoutASessionIsUnauthenticated(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	anonymous := syncclient.NewAccountClient(e.address, http.DefaultClient, func() string { return "" })

	_, err := anonymous.GetAccountChain(t.Context())

	if !isUnauthenticated(err) {
		t.Errorf("error = %v, want unauthenticated", err)
	}
}

type chainRead struct {
	id        []byte
	root      []byte
	signature []byte
}

func readChain(e *env, signer identity.RecoveryKey, root []byte) chainRead {
	e.tb.Helper()
	client := syncclient.NewSignInClient(e.address, http.DefaultClient, machineClock{}.Now)
	challenge, err := client.GetChainReadChallenge(e.tb.Context(), root)
	if err != nil {
		e.tb.Fatal(err)
	}
	signature := signer.SignForRecovery(syncprotocol.SignInPluginID, challenge.Message)
	return chainRead{id: challenge.ID, root: root, signature: signature[:]}
}

func (c chainRead) ask(e *env) ([]byte, bool, error) {
	anonymous := syncclient.NewAccountClient(e.address, http.DefaultClient, func() string { return "" })
	return anonymous.GetAccountChainByRoot(e.tb.Context(), c.id, c.root, c.signature)
}

func TestGetAccountChainByRoot_ReturnsTheStoredChainForASignatureOfTheRecoveryKeyWithoutASession(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	root := e.account.root.Public().Sign

	chain, found, err := readChain(e, e.account.root, root).ask(e)

	if err != nil || !found || !bytes.Equal(chain, e.account.chain.Bytes()) {
		t.Errorf("found = %v, chain of %d bytes, error = %v, want the stored chain", found, len(chain), err)
	}
}

func TestGetAccountChainByRoot_AnswersNotFoundForARootThatNoAccountHas(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	other := newOwnedAccount(t)
	root := other.root.Public().Sign

	chain, found, err := readChain(e, other.root, root).ask(e)

	if err != nil || found || len(chain) != 0 {
		t.Errorf("found = %v, chain of %d bytes, error = %v, want not found with no error", found, len(chain), err)
	}
}

func TestGetAccountChainByRoot_RefusesASignatureThatIsNotOfTheRootKeyWhetherTheAccountExistsOrNot(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	other := newOwnedAccount(t)

	known := readChain(e, other.root, e.account.root.Public().Sign)
	unknown := readChain(e, e.account.root, other.root.Public().Sign)

	for name, read := range map[string]chainRead{"a known root": known, "an unknown root": unknown} {
		_, found, err := read.ask(e)
		if !isUnauthenticated(err) || found {
			t.Errorf("%s: found = %v, error = %v, want unauthenticated", name, found, err)
		}
	}
}

func TestGetAccountChainByRoot_RefusesASignatureOverAnotherPluginDomain(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	root := e.account.root.Public().Sign
	read := readChain(e, e.account.root, root)
	client := syncclient.NewSignInClient(e.address, http.DefaultClient, machineClock{}.Now)
	challenge, err := client.GetChainReadChallenge(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	other := e.account.root.SignForRecovery("md.numen.tools.link", challenge.Message)
	read.id, read.signature = challenge.ID, other[:]

	_, found, err := read.ask(e)

	if !isUnauthenticated(err) || found {
		t.Errorf("found = %v, error = %v, want unauthenticated", found, err)
	}
}

func TestGetAccountChainByRoot_UsesAChallengeOnce(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	read := readChain(e, e.account.root, e.account.root.Public().Sign)

	if _, found, err := read.ask(e); err != nil || !found {
		t.Fatalf("first read: found = %v, error = %v", found, err)
	}
	_, found, err := read.ask(e)

	if !isUnauthenticated(err) || found {
		t.Errorf("second read: found = %v, error = %v, want unauthenticated", found, err)
	}
}

func TestGetAccountChainByRoot_RefusesASignatureOfTheSignInMessageOfTheSameChallenge(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	root := e.account.root.Public().Sign
	client := syncclient.NewSignInClient(e.address, http.DefaultClient, machineClock{}.Now)
	challenge, err := client.GetChallenge(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	signature := e.account.root.SignForRecovery(syncprotocol.SignInPluginID, challenge.Message)
	anonymous := syncclient.NewAccountClient(e.address, http.DefaultClient, func() string { return "" })

	_, found, got := anonymous.GetAccountChainByRoot(t.Context(), challenge.ID, root, signature[:])

	if !isUnauthenticated(got) || found {
		t.Errorf("found = %v, error = %v, want unauthenticated: a sign-in signature must not read a chain", found, got)
	}
}
