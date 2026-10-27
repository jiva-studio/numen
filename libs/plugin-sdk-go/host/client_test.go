package host_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// answering is a host that answers a few calls, and records who asked.
type answering struct {
	numenv1connect.UnimplementedHostVaultServiceHandler
	code       v1.HostVaultErrorCode
	authHeader string
	unlocked   string
	named      string
	baselines  []*v1.HostFileStat
}

func (a *answering) StatFile(_ context.Context, req *connect.Request[v1.StatFileRequest]) (*connect.Response[v1.StatFileResponse], error) {
	a.authHeader = req.Header().Get("Authorization")
	return connect.NewResponse(&v1.StatFileResponse{
		Error: a.code,
		Stat:  &v1.HostFileStat{Path: req.Msg.GetPath(), Size: 7, ModTime: 1_700_000_000_123_456_789},
	}), nil
}

func (a *answering) GetHostVault(_ context.Context, req *connect.Request[v1.GetHostVaultRequest]) (*connect.Response[v1.GetHostVaultResponse], error) {
	a.named = req.Header().Get("Numen-Vault")
	if a.named == "nobody" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no such vault"))
	}
	return connect.NewResponse(&v1.GetHostVaultResponse{Id: a.named, Name: "N", Path: "/p"}), nil
}

func (a *answering) ListHostVaults(context.Context, *connect.Request[v1.ListHostVaultsRequest]) (*connect.Response[v1.ListHostVaultsResponse], error) {
	return connect.NewResponse(&v1.ListHostVaultsResponse{Vaults: []*v1.HostVaultInfo{{Id: "a", Name: "A", Path: "/a"}, {Id: "b", Name: "B", Path: "/b"}}}), nil
}

func (a *answering) EnsureVaultIndexed(context.Context, *connect.Request[v1.EnsureVaultIndexedRequest]) (*connect.Response[v1.EnsureVaultIndexedResponse], error) {
	return connect.NewResponse(&v1.EnsureVaultIndexedResponse{State: v1.HostIndexState_HOST_INDEX_STATE_FAILED, Reason: "disk"}), nil
}

func (a *answering) WatchMoves(ctx context.Context, _ *connect.Request[v1.WatchMovesRequest], stream *connect.ServerStream[v1.WatchMovesResponse]) error {
	_ = stream.Send(&v1.WatchMovesResponse{})
	_ = stream.Send(&v1.WatchMovesResponse{})
	<-ctx.Done()
	return nil
}

func (a *answering) WriteFile(_ context.Context, req *connect.Request[v1.WriteFileRequest]) (*connect.Response[v1.WriteFileResponse], error) {
	a.baselines = append(a.baselines, req.Msg.GetBaseline())
	return connect.NewResponse(&v1.WriteFileResponse{Error: a.code}), nil
}

func (a *answering) GetLinks(context.Context, *connect.Request[v1.GetLinksRequest]) (*connect.Response[v1.GetLinksResponse], error) {
	return connect.NewResponse(&v1.GetLinksResponse{Links: []*v1.HostLink{{
		From: "a.md", To: "b.md", ToVault: "v", LinkType: "cites", TargetScheme: "name", TargetValue: "B", Label: "see", IsAmbiguous: true,
	}}}), nil
}

func (a *answering) LockVault(context.Context, *connect.Request[v1.LockVaultRequest]) (*connect.Response[v1.LockVaultResponse], error) {
	return connect.NewResponse(&v1.LockVaultResponse{Lock: "L1"}), nil
}

func (a *answering) UnlockVault(_ context.Context, req *connect.Request[v1.UnlockVaultRequest]) (*connect.Response[v1.UnlockVaultResponse], error) {
	a.unlocked = req.Msg.GetLock()
	return connect.NewResponse(&v1.UnlockVaultResponse{}), nil
}

func (a *answering) WatchVault(ctx context.Context, _ *connect.Request[v1.WatchVaultRequest], stream *connect.ServerStream[v1.WatchVaultResponse]) error {
	_ = stream.Send(&v1.WatchVaultResponse{})
	_ = stream.Send(&v1.WatchVaultResponse{Paths: []string{"x.md"}})
	_ = stream.Send(&v1.WatchVaultResponse{IsVaultChanged: true})
	<-ctx.Done()
	return nil
}

func clientTo(t *testing.T, a *answering) *host.Client {
	t.Helper()
	mux := httptest.NewServer(handler(a))
	t.Cleanup(mux.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: mux.URL, Token: "tok"})
}

func TestTheClientPresentsItsTokenAndReadsTheFingerprint(t *testing.T) {
	a := &answering{}

	fp, err := clientTo(t, a).StatFile(context.Background(), "n.md")

	want := domain.FileStat{Path: "n.md", Size: 7, ModTime: time.Unix(0, 1_700_000_000_123_456_789)}
	if err != nil || fp != want || a.authHeader != "Bearer tok" {
		t.Errorf("got %+v (%v) authHeader as %q", fp, err, a.authHeader)
	}
}

func TestEachCodeOfTheHostIsItsOwnError(t *testing.T) {
	cases := map[v1.HostVaultErrorCode]error{
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT:        host.ErrNoVault,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_MISSING:         host.ErrNotExist,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_OUTSIDE:         host.ErrOutside,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_STALE:           host.ErrStale,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS:  host.ErrAlreadyExists,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_HELD_BY_ANOTHER: host.ErrHeldByAnother,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_A_NOTE:      host.ErrNotANote,
	}
	for code, want := range cases {
		_, err := clientTo(t, &answering{code: code}).StatFile(context.Background(), "n.md")
		if !errors.Is(err, want) {
			t.Errorf("code %s: got %v, want %v", code, err, want)
		}
	}
}

func TestAWriteSendsTheBaselineAndReportsStale(t *testing.T) {
	a := &answering{code: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_STALE}
	baseline := domain.FileStat{Path: "n.md", Size: 3, ModTime: time.Unix(5, 6)}

	_, err := clientTo(t, a).Write(context.Background(), "n.md", []byte("abc"), baseline)

	if !errors.Is(err, host.ErrStale) || len(a.baselines) != 1 || a.baselines[0].GetModTime() != baseline.ModTime.UnixNano() {
		t.Errorf("got %v and %v", err, a.baselines)
	}
}

func TestALinkPreservesEverythingTheHostSaidOfIt(t *testing.T) {
	links, err := clientTo(t, &answering{}).GetLinks(context.Background(), "a.md")

	want := domain.ResolvedLink{
		Link:        domain.Link{Target: domain.Address{Scheme: "name", Value: "B"}, Type: "cites", Label: "see"},
		From:        "a.md",
		To:          "b.md",
		ToVault:     "v",
		IsAmbiguous: true,
	}
	if err != nil || len(links) != 1 || links[0] != want {
		t.Errorf("got %+v (%v)", links, err)
	}
}

func TestReleasingTheLockPassesItBack(t *testing.T) {
	a := &answering{}

	release, err := clientTo(t, a).Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	release()

	if a.unlocked != "L1" {
		t.Errorf("unlocked %q", a.unlocked)
	}
}

func TestAWatchBeginsAfterTheFirstMessageAndTellsTheChanges(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	changes, err := clientTo(t, &answering{}).Watch(ctx)
	if err != nil {
		t.Fatal(err)
	}

	first := <-changes
	second := <-changes
	if len(first.Paths) != 1 || first.Paths[0] != "x.md" || !second.IsVaultChanged {
		t.Errorf("got %+v then %+v", first, second)
	}
	cancel()
	for range changes {
	}
}

func TestAVaultNamedByIdentityIsAskedForByHeader(t *testing.T) {
	a := &answering{}
	client := clientTo(t, a)

	got, err := client.With("v2").GetVault(context.Background())
	_, unknown := client.With("nobody").GetVault(context.Background())

	if err != nil || got.ID != "v2" || a.named != "nobody" {
		t.Errorf("got %+v (%v), the header was %q", got, err, a.named)
	}
	if !errors.Is(unknown, host.ErrUnknownVault) {
		t.Errorf("an unknown vault was answered %v", unknown)
	}
	displayed, err := client.GetVault(context.Background())
	if err != nil || a.named != "" || displayed.ID != "" {
		t.Errorf("the shown vault was asked for with the header %q (%+v, %v)", a.named, displayed, err)
	}
}

func TestEveryKnownVaultIsListed(t *testing.T) {
	vaults, err := clientTo(t, &answering{}).GetVaults(context.Background())

	if err != nil || len(vaults) != 2 || vaults[1] != (domain.Vault{ID: "b", Name: "B", Path: "/b"}) {
		t.Errorf("got %+v (%v)", vaults, err)
	}
}

func TestWhereTheIndexExistsReachesWithItsReason(t *testing.T) {
	state, err := clientTo(t, &answering{}).EnsureIndexed(context.Background())

	if err != nil || state != (host.IndexState{Status: host.IndexFailed, Reason: "disk"}) {
		t.Errorf("got %+v (%v)", state, err)
	}
}

func TestAWatchOfMovesBeginsAfterTheFirstMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	moves, err := clientTo(t, &answering{}).WatchMoves(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, open := <-moves; !open {
		t.Error("the watch closed before it told anything")
	}
	cancel()
	for range moves {
	}
}

func TestAFingerprintPreservesWhatSortOfSourceTheFileIs(t *testing.T) {
	a := &answering{}
	baseline := domain.FileStat{Path: "n.md", Kind: domain.KindNote, Size: 3, ModTime: time.Unix(5, 6)}

	_, _ = clientTo(t, a).Write(context.Background(), "n.md", []byte("abc"), baseline)

	if len(a.baselines) != 1 || !a.baselines[0].GetIsNote() {
		t.Errorf("the host was sent %v", a.baselines)
	}
}

func TestGetRequestVaultReadsTheVaultHeader(t *testing.T) {
	header := http.Header{}
	if got := host.GetRequestVault(header); got != "" {
		t.Errorf("vault = %q, want none", got)
	}
	header.Set("Numen-Vault", "v-1")
	if got := host.GetRequestVault(header); got != "v-1" {
		t.Errorf("vault = %q, want v-1", got)
	}
}
