package client_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

type shareStub struct {
	syncv1connect.UnimplementedShareServiceHandler

	mu       sync.Mutex
	headers  []http.Header
	fail     error
	pages    [][]*syncv1.ShareLogEntry
	requests []*syncv1.GetShareLogRequest
	changes  *syncv1.GetShareChangesResponse
	appended *syncv1.AppendShareLogEntryResponse
	shares   []*syncv1.ShareSummary
	urls     []*syncv1.BlobDownload
}

func (s *shareStub) record(header http.Header) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.headers = append(s.headers, header.Clone())
	return s.fail
}

func (s *shareStub) AppendShareLogEntry(_ context.Context, req *connect.Request[syncv1.AppendShareLogEntryRequest]) (*connect.Response[syncv1.AppendShareLogEntryResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(s.appended), nil
}

func (s *shareStub) ListShares(_ context.Context, req *connect.Request[syncv1.ListSharesRequest]) (*connect.Response[syncv1.ListSharesResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.ListSharesResponse{Shares: s.shares}), nil
}

func (s *shareStub) GetShareLog(_ context.Context, req *connect.Request[syncv1.GetShareLogRequest]) (*connect.Response[syncv1.GetShareLogResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copied, isRequest := proto.Clone(req.Msg).(*syncv1.GetShareLogRequest)
	if !isRequest {
		return nil, connect.NewError(connect.CodeInternal, errors.New("the clone is not a share log request"))
	}
	s.requests = append(s.requests, copied)
	page := s.pages[len(s.requests)-1]
	return connect.NewResponse(&syncv1.GetShareLogResponse{Entries: page, HasMore: len(s.requests) < len(s.pages), OwnerChain: []byte("chain of the owner")}), nil
}

func (s *shareStub) GetShareChanges(_ context.Context, req *connect.Request[syncv1.GetShareChangesRequest]) (*connect.Response[syncv1.GetShareChangesResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(s.changes), nil
}

func (s *shareStub) GetShareBlobUrls(_ context.Context, req *connect.Request[syncv1.GetShareBlobUrlsRequest]) (*connect.Response[syncv1.GetShareBlobUrlsResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.GetShareBlobUrlsResponse{Downloads: s.urls}), nil
}

func serveShareStub(tb testing.TB, serverVersion uint32) (*shareStub, string) {
	tb.Helper()
	stub := &shareStub{}
	mux := http.NewServeMux()
	mux.Handle(syncv1connect.NewShareServiceHandler(stub, connect.WithInterceptors(syncprotocol.NewVersionHandlerInterceptor(serverVersion))))
	server := httptest.NewServer(mux)
	tb.Cleanup(server.Close)
	return stub, server.URL
}

func newShareClient(address string) *syncclient.ShareClient {
	return syncclient.NewShareClient(address, http.DefaultClient, func() string { return "token-of-the-session" })
}

func newLogEntry(version uint64) *syncv1.ShareLogEntry {
	return &syncv1.ShareLogEntry{ShareRootId: bytes.Repeat([]byte{1}, 16), Version: version}
}

func newDetailedError(tb testing.TB, code connect.Code, detail proto.Message) error {
	tb.Helper()
	failure := connect.NewError(code, errors.New("refused"))
	added, err := connect.NewErrorDetail(detail)
	if err != nil {
		tb.Fatal(err)
	}
	failure.AddDetail(added)
	return failure
}

func TestShareClient_SendsTheVersionAndTheBearerOfTheTokenInEachRequest(t *testing.T) {
	t.Parallel()
	stub, address := serveShareStub(t, syncprotocol.Version)
	stub.appended = &syncv1.AppendShareLogEntryResponse{Committed: true}
	stub.pages = [][]*syncv1.ShareLogEntry{{newLogEntry(1)}}
	stub.changes = &syncv1.GetShareChangesResponse{}
	client := newShareClient(address)
	share := bytes.Repeat([]byte{1}, 16)

	_, appendErr := client.AppendShareLogEntry(t.Context(), newLogEntry(1))
	_, listErr := client.ListShares(t.Context())
	_, _, logErr := client.GetShareLog(t.Context(), share, 0)
	_, changesErr := client.GetShareChanges(t.Context(), share, 0, 0)
	_, urlsErr := client.GetShareBlobUrls(t.Context(), share, [][]byte{bytes.Repeat([]byte{2}, 16)})

	if err := errors.Join(appendErr, listErr, logErr, changesErr, urlsErr); err != nil {
		t.Fatal(err)
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.headers) != 5 {
		t.Fatalf("the server saw %d requests, want 5", len(stub.headers))
	}
	for i, header := range stub.headers {
		if got := header.Get("Authorization"); got != "Bearer token-of-the-session" {
			t.Errorf("request %d: Authorization = %q, want the bearer of the token", i, got)
		}
		if got, want := header.Get(syncprotocol.HeaderVersion), strconv.FormatUint(uint64(syncprotocol.Version), 10); got != want {
			t.Errorf("request %d: %s = %q, want %q", i, syncprotocol.HeaderVersion, got, want)
		}
	}
}

func TestAppendShareLogEntry_ReturnsTheHeadOfARefusedEntry(t *testing.T) {
	t.Parallel()
	stub, address := serveShareStub(t, syncprotocol.Version)
	stub.appended = &syncv1.AppendShareLogEntryResponse{Committed: false, Head: newLogEntry(5)}

	got, err := newShareClient(address).AppendShareLogEntry(t.Context(), newLogEntry(5))
	if err != nil {
		t.Fatal(err)
	}

	if got.GetCommitted() || got.GetHead().GetVersion() != 5 {
		t.Errorf("response = %v, want not committed with the head at version 5", got)
	}
}

func TestListShares_ReturnsTheSharesOfTheAccount(t *testing.T) {
	t.Parallel()
	stub, address := serveShareStub(t, syncprotocol.Version)
	stub.shares = []*syncv1.ShareSummary{{ShareRootId: bytes.Repeat([]byte{1}, 16), HeadVersion: 3, Epoch: 1}}

	got, err := newShareClient(address).ListShares(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0].GetHeadVersion() != 3 || got[0].GetEpoch() != 1 {
		t.Errorf("shares = %v, want the one share at version 3 and epoch 1", got)
	}
}

func TestGetShareLog_ReadsEveryPageAfterTheVersionAndReturnsTheChainOfTheOwner(t *testing.T) {
	t.Parallel()
	stub, address := serveShareStub(t, syncprotocol.Version)
	stub.pages = [][]*syncv1.ShareLogEntry{{newLogEntry(2), newLogEntry(3)}, {newLogEntry(4)}}
	share := bytes.Repeat([]byte{1}, 16)

	entries, chain, err := newShareClient(address).GetShareLog(t.Context(), share, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 3 || entries[0].GetVersion() != 2 || entries[2].GetVersion() != 4 {
		t.Errorf("entries = %v, want versions 2, 3 and 4", entries)
	}
	if string(chain) != "chain of the owner" {
		t.Errorf("chain = %q, want the chain the server sent", chain)
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.requests) != 2 || stub.requests[0].GetSinceVersion() != 1 || stub.requests[1].GetSinceVersion() != 3 ||
		!bytes.Equal(stub.requests[1].GetShareRootId(), share) {
		t.Errorf("requests = %v, want one after version 1 and one after version 3 for the share", stub.requests)
	}
}

func TestGetShareChanges_ReturnsThePageWithItsCursor(t *testing.T) {
	t.Parallel()
	stub, address := serveShareStub(t, syncprotocol.Version)
	stub.changes = &syncv1.GetShareChangesResponse{
		Changes: []*syncv1.ShareChange{{EncMetaHash: bytes.Repeat([]byte{3}, 32), Left: true}}, NextSeq: 9, HasMore: true,
	}

	got, err := newShareClient(address).GetShareChanges(t.Context(), bytes.Repeat([]byte{1}, 16), 4, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.GetChanges()) != 1 || !got.GetChanges()[0].GetLeft() || got.GetNextSeq() != 9 || !got.GetHasMore() {
		t.Errorf("page = %v, want one change that left, next 9, more", got)
	}
}

func TestGetShareBlobUrls_ReturnsOneDownloadForEachId(t *testing.T) {
	t.Parallel()
	stub, address := serveShareStub(t, syncprotocol.Version)
	stub.urls = []*syncv1.BlobDownload{{Found: true, Url: "http://blobs.example/1"}, {Found: false}}

	got, err := newShareClient(address).GetShareBlobUrls(t.Context(), bytes.Repeat([]byte{1}, 16), [][]byte{{1}, {2}})
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || !got[0].GetFound() || got[1].GetFound() {
		t.Errorf("downloads = %v, want one found and one not", got)
	}
}

func TestShareClient_MapsServerErrors(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		err  error
		want error
	}{
		"no share":       {err: connect.NewError(connect.CodeNotFound, errors.New("no")), want: syncprotocol.ErrNoShare},
		"share exists":   {err: connect.NewError(connect.CodeAlreadyExists, errors.New("no")), want: syncprotocol.ErrShareExists},
		"no entitlement": {err: connect.NewError(connect.CodePermissionDenied, errors.New("no")), want: syncprotocol.ErrWriteNotAllowed},
		"unauthorized":   {err: connect.NewError(connect.CodeUnauthenticated, errors.New("no")), want: syncprotocol.ErrUnauthenticated},
	}
	for name, c := range cases {
		stub, address := serveShareStub(t, syncprotocol.Version)
		stub.fail = c.err
		_, err := newShareClient(address).ListShares(t.Context())
		if !errors.Is(err, c.want) {
			t.Errorf("%s: error = %v, want %v", name, err, c.want)
		}
	}
}

func TestShareClient_ReadsTheVersionOfTheEntryThatEndedAShare(t *testing.T) {
	t.Parallel()
	stub, address := serveShareStub(t, syncprotocol.Version)
	stub.fail = newDetailedError(t, connect.CodePermissionDenied, &syncv1.ShareEnded{Version: 7})

	_, err := newShareClient(address).GetShareChanges(t.Context(), bytes.Repeat([]byte{1}, 16), 0, 0)

	var ended syncprotocol.ShareEndedError
	if !errors.As(err, &ended) || ended.Version != 7 {
		t.Errorf("error = %v, want ShareEndedError at version 7", err)
	}
	if errors.Is(err, syncprotocol.ErrWriteNotAllowed) {
		t.Error("an ended share reads as a missing entitlement")
	}
}

func TestShareClient_ReadsTheTimeToWaitOfARateLimit(t *testing.T) {
	t.Parallel()
	stub, address := serveShareStub(t, syncprotocol.Version)
	stub.fail = newDetailedError(t, connect.CodeResourceExhausted, &syncv1.RateLimited{RetryAfterSeconds: 90})

	_, err := newShareClient(address).GetShareBlobUrls(t.Context(), bytes.Repeat([]byte{1}, 16), [][]byte{{1}})

	var limited syncprotocol.RateLimitedError
	if !errors.As(err, &limited) || limited.RetryAfter != 90*time.Second {
		t.Errorf("error = %v, want RateLimitedError after 90 seconds", err)
	}
}

func TestShareClient_RefusesAServerOfAnotherVersion(t *testing.T) {
	t.Parallel()
	_, address := serveShareStub(t, syncprotocol.Version+1)

	_, err := newShareClient(address).ListShares(t.Context())

	if !errors.Is(err, syncprotocol.ErrUpdateRequired) {
		t.Errorf("error = %v, want ErrUpdateRequired", err)
	}
}

func TestShareClient_ReadsARefusedAddReaderWithoutSayingWhy(t *testing.T) {
	t.Parallel()
	stub, address := serveShareStub(t, syncprotocol.Version)
	stub.fail = connect.NewError(connect.CodeInvalidArgument, errors.New("no"))
	entry := &syncv1.ShareLogEntry{Kind: syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER}

	_, err := newShareClient(address).AppendShareLogEntry(t.Context(), entry)

	if !errors.Is(err, syncprotocol.ErrReaderRefused) {
		t.Errorf("error = %v, want ErrReaderRefused", err)
	}
	entry.Kind = syncv1.ShareLogKind_SHARE_LOG_KIND_UPDATE
	if _, err := newShareClient(address).AppendShareLogEntry(t.Context(), entry); errors.Is(err, syncprotocol.ErrReaderRefused) {
		t.Errorf("an update that the server refused reads as a refused reader: %v", err)
	}
}
