package fakeserver_test

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

type machineClock struct{}

func (machineClock) Now() time.Time { return time.Now() }

type ownedAccount struct {
	root  identity.RecoveryKey
	keys  identity.AccountKeys
	chain identity.Chain
}

func newOwnedAccount(tb testing.TB) ownedAccount {
	tb.Helper()
	words, err := identity.NewRecoveryWords(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	root, err := words.DeriveRecoveryKey()
	if err != nil {
		tb.Fatal(err)
	}
	keys, err := identity.NewAccountKeys(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	chain, err := identity.NewChain(root, keys, machineClock{}.Now())
	if err != nil {
		tb.Fatal(err)
	}
	return ownedAccount{root: root, keys: keys, chain: chain}
}

func (a *ownedAccount) addDevice(tb testing.TB, name string) identity.DeviceKey {
	tb.Helper()
	device, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	chain, err := a.chain.AppendDevice(a.root, device.Public(), name, machineClock{}.Now())
	if err != nil {
		tb.Fatal(err)
	}
	a.chain = chain
	return device
}

func newServer(tb testing.TB) (*fakeserver.Server, string) {
	tb.Helper()
	server := fakeserver.New(fakeserver.Config{})
	served := httptest.NewServer(server)
	tb.Cleanup(served.Close)
	server.SetAudience(served.URL)
	return server, served.URL
}

func signIn(tb testing.TB, address string, device identity.DeviceKey, chain identity.Chain) string {
	tb.Helper()
	client := syncclient.NewSignInClient(address, http.DefaultClient, machineClock{}.Now)
	key := device.Public().Sign
	challenge, err := client.GetChallenge(tb.Context(), key)
	if err != nil {
		tb.Fatal(err)
	}
	signature := device.SignForPlugin(syncprotocol.SignInPluginID, challenge.Message)
	session, err := client.SignIn(tb.Context(), syncclient.Proof{ChallengeID: challenge.ID, DeviceSignKey: key, Signature: signature[:], Chain: chain.Bytes()})
	if err != nil {
		tb.Fatal(err)
	}
	return session.Token
}

type env struct {
	tb       testing.TB
	address  string
	account  *ownedAccount
	device   identity.DeviceKey
	token    string
	catalog  *syncclient.CatalogClient
	revision *syncclient.RevisionClient
	blob     *syncclient.BlobClient
	accounts *syncclient.AccountClient

	version uint64
	head    [32]byte
}

func newEnvOn(tb testing.TB, address string, account *ownedAccount, device identity.DeviceKey) *env {
	tb.Helper()
	token := signIn(tb, address, device, account.chain)
	e := &env{tb: tb, address: address, account: account, device: device, token: token}
	provide := func() string { return e.token }
	e.catalog = syncclient.NewCatalogClient(address, http.DefaultClient, provide)
	e.revision = syncclient.NewRevisionClient(address, http.DefaultClient, provide)
	e.blob = syncclient.NewBlobClient(address, http.DefaultClient, provide)
	e.accounts = syncclient.NewAccountClient(address, http.DefaultClient, provide)
	return e
}

func newEnv(tb testing.TB) *env {
	tb.Helper()
	_, address := newServer(tb)
	account := newOwnedAccount(tb)
	device := account.addDevice(tb, "First")
	return newEnvOn(tb, address, &account, device)
}

func newSecondAccountOn(tb testing.TB, address string) *env {
	tb.Helper()
	account := newOwnedAccount(tb)
	device := account.addDevice(tb, "Other")
	return newEnvOn(tb, address, &account, device)
}

func (e *env) signCatalogEntry(entry *syncv1.CatalogEntry) *syncv1.CatalogEntry {
	signature := e.device.SignForPlugin(syncprotocol.SignInPluginID, syncprotocol.GetCatalogEntryMessage(entry))
	entry.Signature = signature[:]
	return entry
}

func (e *env) newCatalogEntry(vault []byte, creates bool, body string) *syncv1.CatalogEntry {
	entry := &syncv1.CatalogEntry{
		Version: e.version + 1, VaultId: vault, CreatesVault: creates, EncBody: []byte(body),
		AuthorDeviceSignKey: e.device.Public().Sign,
	}
	if e.version > 0 {
		entry.PrevHash = bytes.Clone(e.head[:])
	}
	return e.signCatalogEntry(entry)
}

func (e *env) appendEntry(vault []byte, creates bool, body string) *syncv1.AppendCatalogEntryResponse {
	e.tb.Helper()
	entry := e.newCatalogEntry(vault, creates, body)
	response, err := e.catalog.AppendCatalogEntry(e.tb.Context(), entry)
	if err != nil {
		e.tb.Fatal(err)
	}
	if !response.GetCommitted() {
		e.tb.Fatalf("entry %d was not committed", entry.GetVersion())
	}
	e.version = entry.GetVersion()
	e.head = syncprotocol.GetCatalogEntryID(entry)
	return response
}

func (e *env) newVault() []byte {
	e.tb.Helper()
	vault := newID(16)
	e.appendEntry(vault, true, "vault")
	return vault
}

func hashOf(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

func (e *env) upload(vault, body []byte) *syncv1.BlobUpload {
	e.tb.Helper()
	uploads, err := e.blob.ReserveBlobs(e.tb.Context(), vault, []uint64{uint64(len(body))})
	if err != nil {
		e.tb.Fatal(err)
	}
	if len(uploads) != 1 {
		e.tb.Fatalf("uploads = %d, want 1", len(uploads))
	}
	if err := syncclient.PutBlob(e.tb.Context(), http.DefaultClient, uploads[0].GetUrl(), body); err != nil {
		e.tb.Fatal(err)
	}
	return uploads[0]
}

func (e *env) reserveOnly(vault []byte, size int) *syncv1.BlobUpload {
	e.tb.Helper()
	uploads, err := e.blob.ReserveBlobs(e.tb.Context(), vault, []uint64{uint64(size)})
	if err != nil {
		e.tb.Fatal(err)
	}
	return uploads[0]
}

func (e *env) signRevision(revision *syncv1.Revision) *syncv1.Revision {
	signature := e.device.SignForPlugin(syncprotocol.SignInPluginID, syncprotocol.GetRevisionMessage(revision))
	revision.Signature = signature[:]
	return revision
}

func (e *env) newRevisionOf(vault, file, parent []byte, path string, upload *syncv1.BlobUpload, body []byte) *syncv1.Revision {
	return e.signRevision(&syncv1.Revision{
		VaultId: vault, FileId: file, ParentRev: parent,
		BlobId: upload.GetBlobId(), BlobHash: hashOf(body), BlobSize: uint64(len(body)),
		EncMeta: []byte("meta of " + path), PathHash: hashOf([]byte(path)),
		AuthorDeviceSignKey: e.device.Public().Sign,
	})
}

func (e *env) newFile(vault, file, parent []byte, path string, body []byte) *syncv1.Revision {
	e.tb.Helper()
	return e.newRevisionOf(vault, file, parent, path, e.upload(vault, body), body)
}

func (e *env) commit(vault []byte, revisions ...*syncv1.Revision) *syncv1.CommitRevisionsResponse {
	e.tb.Helper()
	response, err := e.revision.CommitRevisions(e.tb.Context(), vault, revisions)
	if err != nil {
		e.tb.Fatal(err)
	}
	return response
}

func (e *env) readAllChanges(vault []byte) []*syncv1.StoredRevision {
	e.tb.Helper()
	response, err := e.revision.GetChanges(e.tb.Context(), vault, 0, 1000)
	if err != nil {
		e.tb.Fatal(err)
	}
	return response.GetChanges()
}

func isUnauthenticated(err error) bool {
	return errors.Is(err, syncprotocol.ErrUnauthenticated) || connect.CodeOf(err) == connect.CodeUnauthenticated
}

func newID(size int) []byte {
	return []byte(rand.Text())[:size]
}
