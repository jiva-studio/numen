package cloud_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/synctest"
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

func TestCloud_WatchKeepsAStreamAliveLongerThanTheTimeoutOfTheRequestClient(t *testing.T) {
	served := httptest.NewServer(newSlowWatchHandler(500 * time.Millisecond))
	t.Cleanup(served.Close)
	hints := cloud.New(cloud.Config{
		Address: served.URL, HTTP: &http.Client{Timeout: 100 * time.Millisecond}, Stream: &http.Client{},
		Token: func() string { return "t" }, Refresh: nil,
	})
	watch, err := hints.Watch(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := watch.Close(); err != nil {
			t.Error(err)
		}
	})

	event, err := watch.Receive(t.Context())

	if err != nil {
		t.Fatal(err)
	}
	if event.Seq != 4 || event.IsHeartbeat() {
		t.Errorf("event = %+v, want the hint at seq 4", event)
	}
}

func TestCloud_WatchReturnsTheHeartbeatTheServerAnnouncedAndTheVaultIDAsTheLinkWritesIt(t *testing.T) {
	c, _ := newWorld(t, fakeserver.Config{Heartbeat: 2 * time.Second})
	c.AddVault(t, "Notes")
	c.Connect(t, "Notes")
	c.WriteNote("Notes", "Shopping list.md", "milk")
	if err := c.Sync(t.Context(), "Notes"); err != nil {
		t.Fatal(err)
	}

	watch, err := c.Cloud.Watch(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := watch.Close(); err != nil {
			t.Error(err)
		}
	})
	event, err := watch.Receive(t.Context())

	if err != nil {
		t.Fatal(err)
	}
	if got := watch.GetHeartbeat(); got != 2*time.Second {
		t.Errorf("heartbeat = %v, want 2s", got)
	}
	if want := c.Link(t, "Notes").RemoteVaultID; event.RemoteVaultID != want || event.Seq != 1 {
		t.Errorf("event = %+v, want vault %s at seq 1", event, want)
	}
}

func TestCloud_WatchReturnsTheErrorOfItsContextFromReceive(t *testing.T) {
	c, _ := newWorld(t, fakeserver.Config{Heartbeat: 2 * time.Second})
	watch, err := c.Cloud.Watch(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := watch.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = watch.Receive(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestCloud_WatchSignsInOnceAndOpensAgainAfterAnUnauthenticatedOpen(t *testing.T) {
	c, server := newWorld(t, fakeserver.Config{})
	server.Fake.EndSessions(nil)
	var refreshes int
	hints := c.NewCloud(c.Server.URL, func(ctx context.Context) error {
		refreshes++
		return c.SignIn(ctx)
	})

	watch, err := hints.Watch(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := watch.Close(); err != nil {
			t.Error(err)
		}
	})

	if refreshes != 1 {
		t.Errorf("sign-ins = %d, want 1", refreshes)
	}
}

type slowWatch struct {
	syncv1connect.UnimplementedWatchServiceHandler

	delay time.Duration
}

func newSlowWatchHandler(delay time.Duration) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(syncv1connect.NewWatchServiceHandler(slowWatch{delay: delay}))
	return mux
}

func (s slowWatch) WatchChanges(ctx context.Context, _ *connect.Request[syncv1.WatchChangesRequest], stream *connect.ServerStream[syncv1.WatchChangesResponse]) error {
	started := &syncv1.WatchChangesResponse{Event: &syncv1.WatchChangesResponse_Started{Started: &syncv1.Started{HeartbeatSeconds: 1}}}
	if err := stream.Send(started); err != nil {
		return err
	}
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return ctx.Err()
	}
	hint := &syncv1.WatchChangesResponse{Event: &syncv1.WatchChangesResponse_Hint{Hint: &syncv1.ChangeHint{VaultId: []byte("vault-of-16-byte"), Seq: 4}}}
	return stream.Send(hint)
}
