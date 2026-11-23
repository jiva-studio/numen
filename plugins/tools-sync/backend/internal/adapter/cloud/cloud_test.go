package cloud_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

func newWorld(t *testing.T, cfg fakeserver.Config) (*synctest.Computer, *synctest.Server) {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, cfg)
	return synctest.NewFirstComputer(t, server, clock, "Test", synctest.NewWords(t)), server
}

func newVaultID(t *testing.T) []byte {
	t.Helper()
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		t.Fatal(err)
	}
	return id
}

func newCloud(c *synctest.Computer, token string, refresh func(context.Context) error) *cloud.Cloud {
	return cloud.New(cloud.Config{Address: c.Server.URL, HTTP: http.DefaultClient, Token: func() string { return token }, Refresh: refresh})
}

func TestCloud_AppendsACatalogEntryAndReadsItBack(t *testing.T) {
	c, _ := newWorld(t, fakeserver.Config{})
	entry := c.NewCatalogEntry(t, nil, newVaultID(t), true, []byte("sealed body"))

	response, err := c.Cloud.AppendCatalogEntry(t.Context(), entry)
	if err != nil {
		t.Fatal(err)
	}

	if !response.GetCommitted() {
		t.Fatal("the entry was not committed")
	}
	entries, err := c.Cloud.GetCatalog(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].GetVersion() != 1 || !bytes.Equal(entries[0].GetEncBody(), []byte("sealed body")) {
		t.Errorf("catalog = %v, want the one entry appended", entries)
	}
}

func TestCloud_ReservesABlobAndUploadsItsBodyToTheURL(t *testing.T) {
	c, _ := newWorld(t, fakeserver.Config{})
	vault := newVaultID(t)
	if _, err := c.Cloud.AppendCatalogEntry(t.Context(), c.NewCatalogEntry(t, nil, vault, true, []byte("sealed body"))); err != nil {
		t.Fatal(err)
	}

	uploads, err := c.Cloud.ReserveBlobs(t.Context(), vault, []uint64{4096})
	if err != nil {
		t.Fatal(err)
	}

	if len(uploads) != 1 || uploads[0].GetSize() != 4096 || uploads[0].GetUrl() == "" {
		t.Fatalf("uploads = %v, want one of 4096 bytes with a URL", uploads)
	}
	if err := c.Cloud.PutBlob(t.Context(), uploads[0].GetUrl(), make([]byte, 4096)); err != nil {
		t.Errorf("PutBlob of the reserved size: %v", err)
	}
	if err := c.Cloud.PutBlob(t.Context(), uploads[0].GetUrl(), make([]byte, 10)); err == nil {
		t.Error("PutBlob of another size succeeded, want an error")
	}
}

func TestCloud_ReadsTheChainOfTheAccount(t *testing.T) {
	c, _ := newWorld(t, fakeserver.Config{})

	chain, err := c.Cloud.GetAccountChain(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(chain, c.Fake.Public().Chain) {
		t.Error("the chain of the account differs from the chain of the identity that signed in")
	}
}

func TestCloud_SignsInAgainOnceWhenTheServerEndedTheSession(t *testing.T) {
	c, server := newWorld(t, fakeserver.Config{})
	before := server.Fake.CountCalls(syncv1connect.SignInServiceSignInProcedure)
	server.Fake.EndSessions(nil)

	entries, err := c.Cloud.GetCatalog(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 0 {
		t.Errorf("entries = %v, want none", entries)
	}
	if got := server.Fake.CountCalls(syncv1connect.SignInServiceSignInProcedure) - before; got != 1 {
		t.Errorf("sign-ins after the session ended = %d, want 1", got)
	}
}

func TestCloud_ReturnsUnauthenticatedWhenTheCallIsRefusedAfterOneSignIn(t *testing.T) {
	c, _ := newWorld(t, fakeserver.Config{})
	var refreshed int
	stale := newCloud(c, "not a session", func(context.Context) error {
		refreshed++
		return nil
	})

	_, err := stale.GetCatalog(t.Context(), 0)

	if !errors.Is(err, syncprotocol.ErrUnauthenticated) {
		t.Errorf("error = %v, want ErrUnauthenticated", err)
	}
	if refreshed != 1 {
		t.Errorf("sign-ins asked for = %d, want 1", refreshed)
	}
}

func TestCloud_ReturnsTheErrorOfASignInThatFails(t *testing.T) {
	c, _ := newWorld(t, fakeserver.Config{})
	refused := errors.New("the sign-in was refused")
	stale := newCloud(c, "not a session", func(context.Context) error { return refused })

	_, err := stale.GetCatalog(t.Context(), 0)

	if !errors.Is(err, refused) {
		t.Errorf("error = %v, want the error of the sign-in", err)
	}
}

func TestCloud_ReturnsUnreachableWithoutSigningInWhenTheServerIsDown(t *testing.T) {
	c, server := newWorld(t, fakeserver.Config{})
	var refreshed int
	counting := newCloud(c, "token", func(context.Context) error {
		refreshed++
		return nil
	})
	server.SetDown(true)

	_, err := counting.GetCatalog(t.Context(), 0)

	if !errors.Is(err, syncprotocol.ErrUnreachable) {
		t.Errorf("error = %v, want ErrUnreachable", err)
	}
	if refreshed != 0 {
		t.Errorf("sign-ins asked for = %d, want none for an unreachable server", refreshed)
	}
}

func TestCloud_ReturnsTheQuotaTheServerRefusedOn(t *testing.T) {
	c, _ := newWorld(t, fakeserver.Config{QuotaBytes: 5000})
	vault := newVaultID(t)
	if _, err := c.Cloud.AppendCatalogEntry(t.Context(), c.NewCatalogEntry(t, nil, vault, true, []byte("sealed body"))); err != nil {
		t.Fatal(err)
	}

	_, err := c.Cloud.ReserveBlobs(t.Context(), vault, []uint64{4096, 4096})

	var quota syncprotocol.QuotaExceededError
	if !errors.As(err, &quota) || quota.Limit != 5000 {
		t.Errorf("error = %v, want QuotaExceededError with the limit of 5000", err)
	}
}

func TestCloud_ReadsNoChangesOfANewVaultAndRejectsAnInvalidRevision(t *testing.T) {
	c, _ := newWorld(t, fakeserver.Config{})
	vault := newVaultID(t)
	if _, err := c.Cloud.AppendCatalogEntry(t.Context(), c.NewCatalogEntry(t, nil, vault, true, []byte("sealed body"))); err != nil {
		t.Fatal(err)
	}

	page, err := c.Cloud.GetChanges(t.Context(), vault, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Cloud.CommitRevisions(t.Context(), vault, []*syncv1.Revision{{VaultId: vault}}); err == nil {
		t.Error("CommitRevisions of an invalid revision succeeded, want an error")
	}

	if len(page.GetChanges()) != 0 || page.GetHasMore() {
		t.Errorf("changes of a new vault = %v, want none", page)
	}
}

func TestCloud_ReadsTheRestoreEpochAndTheKeyThatSignsIt(t *testing.T) {
	c, server := newWorld(t, fakeserver.Config{})
	id, err := server.Fake.IssueRestoreEpoch()
	if err != nil {
		t.Fatal(err)
	}

	epoch, key, err := c.Cloud.GetRestoreEpoch(t.Context())

	if err != nil {
		t.Fatal(err)
	}
	if epoch == nil || !bytes.Equal(epoch.GetId(), id) {
		t.Fatalf("epoch = %v, want id %x", epoch, id)
	}
	if err := syncprotocol.CheckRestoreEpoch(key, epoch); err != nil {
		t.Errorf("CheckRestoreEpoch = %v, want nil", err)
	}
}

func TestCloud_AnswersABlobTheServerLostAsBlobMissing(t *testing.T) {
	c, server := newWorld(t, fakeserver.Config{})
	c.Connect(t, "Test")
	c.WriteNote("Test", "Shopping list.md", "milk")
	if err := c.Sync(t.Context(), "Test"); err != nil {
		t.Fatal(err)
	}
	remote := c.GetRemoteID(t, "Test")
	head := c.ReadChanges(t, remote)[0].GetRevision()
	if err := server.Fake.DropBlob(head.GetBlobId()); err != nil {
		t.Fatal(err)
	}

	_, err := c.Cloud.GetBlobUrls(t.Context(), remote, [][]byte{head.GetBlobId()})

	if !errors.Is(err, syncprotocol.ErrBlobMissing) {
		t.Errorf("GetBlobUrls = %v, want ErrBlobMissing", err)
	}
}
