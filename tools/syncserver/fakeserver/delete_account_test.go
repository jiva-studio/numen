package fakeserver_test

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

func (e *env) deleteAccount(signature func(message []byte) []byte) (time.Time, error) {
	e.tb.Helper()
	signIn := syncclient.NewSignInClient(e.address, http.DefaultClient, machineClock{}.Now)
	challenge, err := signIn.GetAccountDeleteChallenge(e.tb.Context(), e.device.Public().Sign)
	if err != nil {
		e.tb.Fatal(err)
	}
	return e.accounts.DeleteAccount(e.tb.Context(), challenge.ID, signature(challenge.Message))
}

func (e *env) signDelete(message []byte) []byte {
	signature := e.device.SignForPlugin(syncprotocol.SignInPluginID, message)
	return signature[:]
}

func (e *env) signInAgain(restore bool) error {
	e.tb.Helper()
	client := syncclient.NewSignInClient(e.address, http.DefaultClient, machineClock{}.Now)
	key := e.device.Public().Sign
	challenge, err := client.GetChallenge(e.tb.Context(), key)
	if err != nil {
		e.tb.Fatal(err)
	}
	signature := e.device.SignForPlugin(syncprotocol.SignInPluginID, challenge.Message)
	session, err := client.SignIn(e.tb.Context(), syncclient.Proof{
		ChallengeID: challenge.ID, DeviceSignKey: key, Signature: signature[:], Chain: e.account.chain.Bytes(), RestoreAccount: restore,
	})
	if err == nil {
		e.token = session.Token
	}
	return err
}

func TestDeleteAccount_RefusesASignatureMadeForSigningIn(t *testing.T) {
	t.Parallel()
	e := newEnv(t)

	_, err := e.deleteAccount(func(message []byte) []byte {
		signature := e.device.SignForPlugin(syncprotocol.SignInPluginID, append([]byte("other"), message...))
		return signature[:]
	})

	if err == nil {
		t.Fatal("the account was deleted with a signature over another message")
	}
	if e.signInAgain(false) != nil {
		t.Error("the account stopped syncing after a refused deletion")
	}
}

func TestDeleteAccount_EndsTheSessionsAndRefusesTheNextSignInWithTheDateOfTheDeletion(t *testing.T) {
	t.Parallel()
	e := newEnv(t)

	deleteAt, err := e.deleteAccount(e.signDelete)

	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.accounts.GetAccountChain(t.Context()); err == nil {
		t.Error("the session of a deleting account still reads")
	}
	deleting, ok := errors.AsType[syncprotocol.AccountDeletingError](e.signInAgain(false))
	if !ok || !deleting.DeleteAt.Equal(deleteAt) {
		t.Errorf("sign-in error = %v, want AccountDeletingError at %v", deleting, deleteAt)
	}
	if got := deleteAt.Sub(machineClock{}.Now()); got < fakeserver.DeletionGrace-time.Minute || got > fakeserver.DeletionGrace {
		t.Errorf("the deletion is in %v, want the grace of %v", got, fakeserver.DeletionGrace)
	}
}

func TestDeleteAccount_IsRestoredByASignInThatAsksForIt(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if _, err := e.deleteAccount(e.signDelete); err != nil {
		t.Fatal(err)
	}

	if err := e.signInAgain(true); err != nil {
		t.Fatalf("restore: %v", err)
	}

	if err := e.signInAgain(false); err != nil {
		t.Errorf("sign-in after the restore: %v", err)
	}
}

func TestDeleteAccount_AfterARestoreStartsANewGraceAndNeverAnEarlierDate(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	first, err := e.deleteAccount(e.signDelete)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.signInAgain(true); err != nil {
		t.Fatal(err)
	}

	second, err := newEnvOn(t, e.address, e.account, e.device).deleteAccount(e.signDelete)

	if err != nil || second.Before(first) {
		t.Errorf("second deletion at %v, error %v, want a date not before %v", second, err, first)
	}
}

func TestExportAccount_ListsTheChainAndTheVaultsOfTheAccountOnly(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	mine := newSecondAccountOn(t, address)
	other := newSecondAccountOn(t, address)
	myVault := make([]byte, 16)
	myVault[0] = 1
	if _, err := mine.catalog.AppendCatalogEntry(t.Context(), mine.newCatalogEntry(myVault, true, "mine")); err != nil {
		t.Fatal(err)
	}
	otherVault := make([]byte, 16)
	otherVault[0] = 2
	if _, err := other.catalog.AppendCatalogEntry(t.Context(), other.newCatalogEntry(otherVault, true, "other")); err != nil {
		t.Fatal(err)
	}

	document, err := mine.accounts.ExportAccount(t.Context())

	if err != nil {
		t.Fatal(err)
	}
	var exported struct {
		Account string `json:"account"`
		Vaults  []struct {
			ID string `json:"id"`
		} `json:"vaults"`
	}
	if err := json.Unmarshal(document, &exported); err != nil {
		t.Fatalf("the document is not JSON: %v", err)
	}
	if want := mine.account.chain.Root().Fingerprint().String(); exported.Account != want {
		t.Errorf("account = %q, want the fingerprint of the account", exported.Account)
	}
	if len(exported.Vaults) != 1 {
		t.Errorf("the document lists %d vaults, want the 1 vault of the account", len(exported.Vaults))
	}
}

func TestDeleteAccount_IsNotRestoredByASignInOfARevokedDevice(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	revoked := e.account.addDevice(t, "Second")
	staleChain := e.account.chain
	signIn(t, e.address, e.device, staleChain)
	chain, keys, err := staleChain.RevokeDevice(e.account.root, e.account.keys, revoked.Public().Sign, rand.Reader, machineClock{}.Now())
	if err != nil {
		t.Fatal(err)
	}
	e.account.chain, e.account.keys = chain, keys
	signIn(t, e.address, e.device, chain)
	if _, err := e.deleteAccount(e.signDelete); err != nil {
		t.Fatal(err)
	}
	client := syncclient.NewSignInClient(e.address, http.DefaultClient, machineClock{}.Now)
	challenge, err := client.GetChallenge(t.Context(), revoked.Public().Sign)
	if err != nil {
		t.Fatal(err)
	}
	signature := revoked.SignForPlugin(syncprotocol.SignInPluginID, challenge.Message)

	_, err = client.SignIn(t.Context(), syncclient.Proof{
		ChallengeID: challenge.ID, DeviceSignKey: revoked.Public().Sign, Signature: signature[:], Chain: staleChain.Bytes(), RestoreAccount: true,
	})

	if err == nil {
		t.Fatal("a revoked device signed in")
	}
	if _, isDeleting := errors.AsType[syncprotocol.AccountDeletingError](e.signInAgain(false)); !isDeleting {
		t.Error("a sign-in of a revoked device canceled the deletion of the account")
	}
}
