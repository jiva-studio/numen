package fakeserver_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

const (
	testHeartbeat = 50 * time.Millisecond
	receiveLimit  = 2 * time.Second
)

type received struct {
	message *syncv1.WatchChangesResponse
	err     error
}

type watcher struct {
	tb     testing.TB
	stream *syncclient.WatchStream
}

func newWatchServer(tb testing.TB, heartbeat time.Duration) (*fakeserver.Server, string) {
	tb.Helper()
	server := fakeserver.New(fakeserver.Config{Heartbeat: heartbeat})
	served := httptest.NewServer(server)
	tb.Cleanup(served.Close)
	server.SetAudience(served.URL)
	return server, served.URL
}

func newWatchEnv(tb testing.TB) (*fakeserver.Server, *env) {
	tb.Helper()
	server, address := newWatchServer(tb, testHeartbeat)
	account := newOwnedAccount(tb)
	device := account.addDevice(tb, "First")
	return server, newEnvOn(tb, address, &account, device)
}

func (e *env) openWatch() *watcher {
	e.tb.Helper()
	client := syncclient.NewWatchClient(e.address, &http.Client{}, func() string { return e.token })
	stream, err := client.WatchChanges(e.tb.Context())
	if err != nil {
		e.tb.Fatal(err)
	}
	e.tb.Cleanup(func() {
		if err := stream.Close(); err != nil {
			e.tb.Error(err)
		}
	})
	return &watcher{tb: e.tb, stream: stream}
}

func (w *watcher) receive() received {
	w.tb.Helper()
	done := make(chan received, 1)
	go func() {
		message, err := w.stream.Receive()
		done <- received{message: message, err: err}
	}()
	select {
	case got := <-done:
		return got
	case <-time.After(receiveLimit):
		w.tb.Fatal("no message within the limit")
		return received{}
	}
}

func (w *watcher) receiveHint() *syncv1.ChangeHint {
	w.tb.Helper()
	for {
		got := w.receive()
		if got.err != nil {
			w.tb.Fatal(got.err)
		}
		if hint := got.message.GetHint(); hint != nil {
			return hint
		}
	}
}

func (w *watcher) receiveHeartbeat() {
	w.tb.Helper()
	got := w.receive()
	if got.err != nil {
		w.tb.Fatal(got.err)
	}
	if got.message.GetHeartbeat() == nil {
		w.tb.Fatalf("message = %v, want a heartbeat", got.message)
	}
}

func (e *env) commitNote(vault []byte, path string) uint64 {
	e.tb.Helper()
	return e.commit(vault, e.newFile(vault, newID(16), nil, path, []byte("text of "+path))).GetSeq()
}

func TestWatchChanges_SendsStartedWithTheAnnouncedHeartbeatFirst(t *testing.T) {
	t.Parallel()
	_, address := newWatchServer(t, 2*time.Second)
	account := newOwnedAccount(t)
	e := newEnvOn(t, address, &account, account.addDevice(t, "First"))

	watch := e.openWatch()

	if got := watch.stream.Heartbeat(); got != 2*time.Second {
		t.Errorf("heartbeat = %v, want 2s", got)
	}
}

func TestWatchChanges_HintsEveryVaultThatHasASeqAfterStartedAtItsSeqNow(t *testing.T) {
	t.Parallel()
	_, e := newWatchEnv(t)
	vault := e.newVault()
	e.commitNote(vault, "one.md")
	e.commitNote(vault, "two.md")
	want := e.commitNote(vault, "three.md")

	hint := e.openWatch().receiveHint()

	if string(hint.GetVaultId()) != string(vault) || hint.GetSeq() != want {
		t.Errorf("hint = %v, want vault %x at seq %d", hint, vault, want)
	}
}

func TestWatchChanges_NeverHintsAVaultWithSeqZero(t *testing.T) {
	t.Parallel()
	_, e := newWatchEnv(t)
	e.newVault()
	watch := e.openWatch()

	watch.receiveHeartbeat()
}

func TestWatchChanges_HintsACommitOfAnotherDeviceWithItsSeq(t *testing.T) {
	t.Parallel()
	_, first := newWatchEnv(t)
	vault := first.newVault()
	second := newEnvOn(t, first.address, first.account, first.account.addDevice(t, "Second"))
	watch := first.openWatch()

	want := second.commitNote(vault, "milk.md")

	if hint := watch.receiveHint(); hint.GetSeq() != want || string(hint.GetVaultId()) != string(vault) {
		t.Errorf("hint = %v, want vault %x at seq %d", hint, vault, want)
	}
}

func TestWatchChanges_NeverHintsALowerSeqAfterAHigherOne(t *testing.T) {
	t.Parallel()
	_, e := newWatchEnv(t)
	vault := e.newVault()
	watch := e.openWatch()
	e.commitNote(vault, "one.md")
	last := e.commitNote(vault, "two.md")

	var seen uint64
	for seen < last {
		hint := watch.receiveHint()
		if hint.GetSeq() < seen {
			t.Fatalf("hint seq %d came after %d", hint.GetSeq(), seen)
		}
		seen = hint.GetSeq()
	}
}

func TestWatchChanges_SendsAHeartbeatOnASilentStream(t *testing.T) {
	t.Parallel()
	_, e := newWatchEnv(t)

	e.openWatch().receiveHeartbeat()
}

func TestWatchChanges_HintsNothingOfAnotherAccount(t *testing.T) {
	t.Parallel()
	_, e := newWatchEnv(t)
	watch := e.openWatch()
	other := newSecondAccountOn(t, e.address)

	other.commitNote(other.newVault(), "secret.md")

	watch.receiveHeartbeat()
}

func TestEndStreams_EndsEveryOpenStreamWithAnEndOfStream(t *testing.T) {
	t.Parallel()
	server, e := newWatchEnv(t)
	watch := e.openWatch()

	server.EndStreams()

	for {
		got := watch.receive()
		if got.err == nil {
			continue
		}
		if !errors.Is(got.err, io.EOF) {
			t.Errorf("error = %v, want io.EOF", got.err)
		}
		return
	}
}

func TestWatchChanges_EndsCleanlyWhenTheSessionEnds(t *testing.T) {
	t.Parallel()
	server, e := newWatchEnv(t)
	watch := e.openWatch()

	server.EndSessions(nil)

	for {
		got := watch.receive()
		if got.err == nil {
			continue
		}
		if !errors.Is(got.err, io.EOF) {
			t.Errorf("error = %v, want io.EOF at the next heartbeat", got.err)
		}
		return
	}
}

func TestWatchChanges_RefusesAnOpenWithoutALiveSessionAsUnauthenticated(t *testing.T) {
	t.Parallel()
	server, e := newWatchEnv(t)
	server.EndSessions(nil)

	_, err := syncclient.NewWatchClient(e.address, &http.Client{}, func() string { return e.token }).WatchChanges(t.Context())

	if !isUnauthenticated(err) {
		t.Errorf("error = %v, want unauthenticated", err)
	}
}

func TestWatchChanges_ServesAnAccountWithoutASubscriptionThatIsReadOnly(t *testing.T) {
	t.Parallel()
	server := fakeserver.New(fakeserver.Config{NeedsSubscription: true, Heartbeat: testHeartbeat})
	served := httptest.NewServer(server)
	t.Cleanup(served.Close)
	server.SetAudience(served.URL)
	account := newOwnedAccount(t)
	e := newEnvOn(t, served.URL, &account, account.addDevice(t, "First"))

	e.openWatch().receiveHeartbeat()
}

func TestCountStreams_CountsOpenStreamsAndDropsAClosedOne(t *testing.T) {
	t.Parallel()
	server, e := newWatchEnv(t)
	first, second := e.openWatch(), e.openWatch()

	if got := server.CountStreams(); got != 2 {
		t.Fatalf("streams = %d, want 2", got)
	}
	if err := first.stream.Close(); err != nil {
		t.Fatal(err)
	}
	second.receiveHeartbeat()
	clock := machineClock{}
	deadline := clock.Now().Add(receiveLimit)
	for server.CountStreams() != 1 && clock.Now().Before(deadline) {
		second.receiveHeartbeat()
	}
	if got := server.CountStreams(); got != 1 {
		t.Errorf("streams = %d, want 1", got)
	}
}

func TestCountCalls_CountsTheOpenedStreamsOfTheWatchProcedure(t *testing.T) {
	t.Parallel()
	server, e := newWatchEnv(t)
	e.openWatch()
	e.openWatch()

	if got := server.CountCalls(syncv1connect.WatchServiceWatchChangesProcedure); got != 2 {
		t.Errorf("opens = %d, want 2", got)
	}
}

func TestAdminStreams_ReportsTheCountAsText(t *testing.T) {
	t.Parallel()
	_, e := newWatchEnv(t)
	e.openWatch()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, e.address+"/admin/streams", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	if got := strings.TrimSpace(string(body)); got != "1" {
		t.Errorf("body = %q, want 1", got)
	}
}

func TestAdminStreamsEnd_EndsEveryStream(t *testing.T) {
	t.Parallel()
	server, e := newWatchEnv(t)
	watch := e.openWatch()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, e.address+"/admin/streams/end", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}

	for {
		got := watch.receive()
		if got.err != nil {
			if !errors.Is(got.err, io.EOF) {
				t.Errorf("error = %v, want io.EOF", got.err)
			}
			break
		}
	}
	if got := server.CountStreams(); got != 0 {
		t.Errorf("streams = %d, want 0", got)
	}
}

func TestRestore_KeepsTheVaultSeqsAndTheCatchUpReportsThem(t *testing.T) {
	t.Parallel()
	server, e := newWatchEnv(t)
	vault := e.newVault()
	want := e.commitNote(vault, "milk.md")
	snapshot, err := server.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored := fakeserver.New(fakeserver.Config{Heartbeat: testHeartbeat})
	if err := restored.Restore(snapshot); err != nil {
		t.Fatal(err)
	}
	served := httptest.NewServer(restored)
	t.Cleanup(served.Close)
	restored.SetAudience(served.URL)
	again := newEnvOn(t, served.URL, e.account, e.device)

	hint := again.openWatch().receiveHint()

	if hint.GetSeq() != want {
		t.Errorf("seq = %d, want %d", hint.GetSeq(), want)
	}
}
