package client_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

type stubServer struct {
	syncv1connect.UnimplementedCatalogServiceHandler
	syncv1connect.UnimplementedRevisionServiceHandler
	syncv1connect.UnimplementedBlobServiceHandler
	syncv1connect.UnimplementedAccountServiceHandler

	mu        sync.Mutex
	headers   []http.Header
	fail      error
	pageCalls []uint64
	chain     []byte
}

func (s *stubServer) record(header http.Header) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.headers = append(s.headers, header.Clone())
	return s.fail
}

func (s *stubServer) calls() []http.Header {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]http.Header(nil), s.headers...)
}

func (s *stubServer) AppendCatalogEntry(_ context.Context, req *connect.Request[syncv1.AppendCatalogEntryRequest]) (*connect.Response[syncv1.AppendCatalogEntryResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.AppendCatalogEntryResponse{Committed: true, Head: req.Msg.GetEntry()}), nil
}

func (s *stubServer) GetCatalog(_ context.Context, req *connect.Request[syncv1.GetCatalogRequest]) (*connect.Response[syncv1.GetCatalogResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.pageCalls = append(s.pageCalls, req.Msg.GetSinceVersion())
	s.mu.Unlock()
	const total, perPage = 5, 2
	response := &syncv1.GetCatalogResponse{}
	for version := req.Msg.GetSinceVersion() + 1; version <= total && len(response.GetEntries()) < perPage; version++ {
		response.Entries = append(response.GetEntries(), &syncv1.CatalogEntry{Version: version})
	}
	last := req.Msg.GetSinceVersion() + uint64(len(response.GetEntries()))
	response.HasMore = last < total
	return connect.NewResponse(response), nil
}

func (s *stubServer) GetVaultUsage(_ context.Context, req *connect.Request[syncv1.GetVaultUsageRequest]) (*connect.Response[syncv1.GetVaultUsageResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.GetVaultUsageResponse{Vaults: []*syncv1.VaultUsage{{VaultId: bytes.Repeat([]byte{1}, 16), UsedBytes: 4096}}}), nil
}

func (s *stubServer) DeleteVault(_ context.Context, req *connect.Request[syncv1.DeleteVaultRequest]) (*connect.Response[syncv1.DeleteVaultResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.DeleteVaultResponse{Deleted: true, FreedBytes: uint64(len(req.Msg.GetVaultId()))}), nil
}

func (s *stubServer) CommitRevisions(_ context.Context, req *connect.Request[syncv1.CommitRevisionsRequest]) (*connect.Response[syncv1.CommitRevisionsResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.CommitRevisionsResponse{Committed: true, Seq: 4}), nil
}

func (s *stubServer) GetChanges(_ context.Context, req *connect.Request[syncv1.GetChangesRequest]) (*connect.Response[syncv1.GetChangesResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.GetChangesResponse{NextSeq: req.Msg.GetSinceSeq() + 1, HeadSeq: 9}), nil
}

func (s *stubServer) ReserveBlobs(_ context.Context, req *connect.Request[syncv1.ReserveBlobsRequest]) (*connect.Response[syncv1.ReserveBlobsResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	response := &syncv1.ReserveBlobsResponse{}
	for _, size := range req.Msg.GetSizes() {
		response.Uploads = append(response.GetUploads(), &syncv1.BlobUpload{BlobId: bytes.Repeat([]byte{4}, 16), Url: "http://upload.example/x", Size: size})
	}
	return connect.NewResponse(response), nil
}

func (s *stubServer) GetBlobUrls(_ context.Context, req *connect.Request[syncv1.GetBlobUrlsRequest]) (*connect.Response[syncv1.GetBlobUrlsResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	response := &syncv1.GetBlobUrlsResponse{}
	for _, id := range req.Msg.GetBlobIds() {
		response.Downloads = append(response.GetDownloads(), &syncv1.BlobDownload{BlobId: id, Url: "http://download.example/x", Found: true})
	}
	return connect.NewResponse(response), nil
}

func (s *stubServer) GetAccountChain(_ context.Context, req *connect.Request[syncv1.GetAccountChainRequest]) (*connect.Response[syncv1.GetAccountChainResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.GetAccountChainResponse{Chain: s.chain}), nil
}

func (s *stubServer) DeleteAccount(_ context.Context, req *connect.Request[syncv1.DeleteAccountRequest]) (*connect.Response[syncv1.DeleteAccountResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.DeleteAccountResponse{DeleteAt: 1_800_000_000}), nil
}

func (s *stubServer) ExportAccount(_ context.Context, req *connect.Request[syncv1.ExportAccountRequest]) (*connect.Response[syncv1.ExportAccountResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.ExportAccountResponse{Document: []byte(`{"vaults":[]}`)}), nil
}

func (s *stubServer) GetAccountChainByRoot(_ context.Context, req *connect.Request[syncv1.GetAccountChainByRootRequest]) (*connect.Response[syncv1.GetAccountChainByRootResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.GetAccountChainByRootResponse{Found: len(s.chain) > 0, Chain: s.chain}), nil
}

func serveStub(tb testing.TB, serverVersion uint32) (*stubServer, string) {
	tb.Helper()
	stub := &stubServer{}
	mux := http.NewServeMux()
	options := connect.WithInterceptors(syncprotocol.NewVersionHandlerInterceptor(serverVersion))
	mux.Handle(syncv1connect.NewCatalogServiceHandler(stub, options))
	mux.Handle(syncv1connect.NewRevisionServiceHandler(stub, options))
	mux.Handle(syncv1connect.NewBlobServiceHandler(stub, options))
	mux.Handle(syncv1connect.NewAccountServiceHandler(stub, options))
	server := httptest.NewServer(mux)
	tb.Cleanup(server.Close)
	return stub, server.URL
}

type stubCalls struct {
	catalog  *syncclient.CatalogClient
	revision *syncclient.RevisionClient
	blob     *syncclient.BlobClient
	account  *syncclient.AccountClient
}

func newStubCalls(address string, token func() string) stubCalls {
	return stubCalls{
		catalog:  syncclient.NewCatalogClient(address, http.DefaultClient, token),
		revision: syncclient.NewRevisionClient(address, http.DefaultClient, token),
		blob:     syncclient.NewBlobClient(address, http.DefaultClient, token),
		account:  syncclient.NewAccountClient(address, http.DefaultClient, token),
	}
}

func (c stubCalls) callAll(ctx context.Context) map[string]error {
	vault := bytes.Repeat([]byte{1}, 16)
	errs := map[string]error{}
	_, errs["AppendCatalogEntry"] = c.catalog.AppendCatalogEntry(ctx, newCatalogEntry())
	_, errs["GetCatalog"] = c.catalog.GetCatalog(ctx, 4)
	_, errs["GetVaultUsage"] = c.catalog.GetVaultUsage(ctx)
	_, errs["DeleteVault"] = c.catalog.DeleteVault(ctx, vault)
	_, errs["CommitRevisions"] = c.revision.CommitRevisions(ctx, vault, []*syncv1.Revision{newRevision()})
	_, errs["GetChanges"] = c.revision.GetChanges(ctx, vault, 0, 0)
	_, errs["ReserveBlobs"] = c.blob.ReserveBlobs(ctx, vault, []uint64{4096})
	_, errs["GetBlobUrls"] = c.blob.GetBlobUrls(ctx, vault, [][]byte{bytes.Repeat([]byte{4}, 16)})
	_, errs["GetAccountChain"] = c.account.GetAccountChain(ctx)
	_, _, errs["GetAccountChainByRoot"] = c.account.GetAccountChainByRoot(ctx, bytes.Repeat([]byte{1}, 16), bytes.Repeat([]byte{3}, 32), bytes.Repeat([]byte{8}, 64))
	_, errs["DeleteAccount"] = c.account.DeleteAccount(ctx, bytes.Repeat([]byte{1}, 16), bytes.Repeat([]byte{8}, 64))
	_, errs["ExportAccount"] = c.account.ExportAccount(ctx)
	return errs
}

func TestEveryClientSendsTheVersionAndTheBearerOfTheTokenInEachRequest(t *testing.T) {
	t.Parallel()
	stub, address := serveStub(t, syncprotocol.Version)

	for name, err := range newStubCalls(address, func() string { return "token-of-the-session" }).callAll(t.Context()) {
		if err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}

	calls := stub.calls()
	if len(calls) != 12 {
		t.Fatalf("the server saw %d requests, want 12", len(calls))
	}
	for i, header := range calls {
		if got := header.Get("Authorization"); got != "Bearer token-of-the-session" {
			t.Errorf("request %d: Authorization = %q, want the bearer of the token", i, got)
		}
		if got := header.Get(syncprotocol.HeaderVersion); got != "2" {
			t.Errorf("request %d: %s = %q, want 2", i, syncprotocol.HeaderVersion, got)
		}
	}
}

func TestEveryClientReadsTheTokenAtEachCall(t *testing.T) {
	t.Parallel()
	stub, address := serveStub(t, syncprotocol.Version)
	var mu sync.Mutex
	token := "first"
	catalog := syncclient.NewCatalogClient(address, http.DefaultClient, func() string {
		mu.Lock()
		defer mu.Unlock()
		return token
	})

	if _, err := catalog.GetCatalog(t.Context(), 4); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	token = "second"
	mu.Unlock()
	if _, err := catalog.GetCatalog(t.Context(), 4); err != nil {
		t.Fatal(err)
	}

	calls := stub.calls()
	if len(calls) < 2 || calls[0].Get("Authorization") != "Bearer first" || calls[len(calls)-1].Get("Authorization") != "Bearer second" {
		t.Errorf("authorization headers = %v, want the first token and then the second", calls)
	}
}

func TestEveryClientReportsAServerOfAnotherVersionAsUpdateRequired(t *testing.T) {
	t.Parallel()
	stub, address := serveStub(t, syncprotocol.Version+1)

	for name, err := range newStubCalls(address, func() string { return "t" }).callAll(t.Context()) {
		if !errors.Is(err, syncprotocol.ErrUpdateRequired) {
			t.Errorf("%s: error = %v, want ErrUpdateRequired", name, err)
		}
	}
	if got := stub.calls(); len(got) != 0 {
		t.Errorf("the server ran %d calls, want none", len(got))
	}
}

func TestTheClientsMapServerFailuresToTheErrorsOfThePackage(t *testing.T) {
	t.Parallel()
	quotaDetail, err := connect.NewErrorDetail(&syncv1.QuotaExceeded{UsedBytes: 7, LimitBytes: 5})
	if err != nil {
		t.Fatal(err)
	}
	notReadyDetail, err := connect.NewErrorDetail(&syncv1.BlobNotReady{RevisionIndex: 3})
	if err != nil {
		t.Fatal(err)
	}
	withDetail := func(code connect.Code, detail *connect.ErrorDetail) error {
		failure := connect.NewError(code, errors.New("refused"))
		failure.AddDetail(detail)
		return failure
	}

	for name, one := range map[string]struct {
		err  error
		want error
	}{
		"unauthenticated":  {err: connect.NewError(connect.CodeUnauthenticated, errors.New("no")), want: syncprotocol.ErrUnauthenticated},
		"unavailable":      {err: connect.NewError(connect.CodeUnavailable, errors.New("down")), want: syncprotocol.ErrUnreachable},
		"deadline":         {err: connect.NewError(connect.CodeDeadlineExceeded, errors.New("slow")), want: syncprotocol.ErrUnreachable},
		"permission":       {err: connect.NewError(connect.CodePermissionDenied, errors.New("no")), want: syncprotocol.ErrWriteNotAllowed},
		"no vault":         {err: connect.NewError(connect.CodeNotFound, errors.New("no")), want: syncprotocol.ErrNoVault},
		"vault exists":     {err: connect.NewError(connect.CodeAlreadyExists, errors.New("no")), want: syncprotocol.ErrVaultExists},
		"invalid revision": {err: connect.NewError(connect.CodeInvalidArgument, errors.New("invalid revision")), want: nil},
		"quota":            {err: withDetail(connect.CodeResourceExhausted, quotaDetail), want: syncprotocol.QuotaExceededError{Used: 7, Limit: 5}},
		"blob not ready":   {err: withDetail(connect.CodeFailedPrecondition, notReadyDetail), want: syncprotocol.BlobNotReadyError{Index: 3}},
	} {
		stub, address := serveStub(t, syncprotocol.Version)
		stub.fail = one.err
		calls := newStubCalls(address, func() string { return "t" })

		for method, got := range calls.callAll(t.Context()) {
			if one.want == nil {
				if got == nil || errors.Is(got, syncprotocol.ErrUnreachable) || errors.Is(got, syncprotocol.ErrUnauthenticated) {
					t.Errorf("%s of %s: error = %v, want the refusal left as the server sent it", method, name, got)
				}
				continue
			}
			if !errors.Is(got, one.want) {
				t.Errorf("%s of %s: error = %v, want %v", method, name, got, one.want)
			}
		}
	}
}

func TestErrQuotaExceededCarriesTheUsedAndTheLimitOfTheServer(t *testing.T) {
	t.Parallel()
	detail, err := connect.NewErrorDetail(&syncv1.QuotaExceeded{UsedBytes: 700, LimitBytes: 500})
	if err != nil {
		t.Fatal(err)
	}
	stub, address := serveStub(t, syncprotocol.Version)
	failure := connect.NewError(connect.CodeResourceExhausted, errors.New("full"))
	failure.AddDetail(detail)
	stub.fail = failure

	_, got := syncclient.NewBlobClient(address, http.DefaultClient, func() string { return "t" }).ReserveBlobs(t.Context(), bytes.Repeat([]byte{1}, 16), []uint64{4096})

	var quota syncprotocol.QuotaExceededError
	if !errors.As(got, &quota) || quota.Used != 700 || quota.Limit != 500 {
		t.Errorf("error = %v, quota = %+v, want used 700 and limit 500", got, quota)
	}
	if notReady, isNotReady := errors.AsType[syncprotocol.BlobNotReadyError](got); isNotReady {
		t.Errorf("a quota refusal reads as a blob that is not ready: %+v", notReady)
	}
}

func TestGetCatalogReadsEveryPageAndReturnsTheEntriesInOrder(t *testing.T) {
	t.Parallel()
	stub, address := serveStub(t, syncprotocol.Version)

	entries, err := syncclient.NewCatalogClient(address, http.DefaultClient, func() string { return "t" }).GetCatalog(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 5 {
		t.Fatalf("entries = %d, want 5", len(entries))
	}
	for i, entry := range entries {
		if entry.GetVersion() != uint64(i+1) {
			t.Errorf("entry %d has version %d, want %d", i, entry.GetVersion(), i+1)
		}
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.pageCalls) != 3 || stub.pageCalls[0] != 0 || stub.pageCalls[1] != 2 || stub.pageCalls[2] != 4 {
		t.Errorf("pages asked after versions %v, want [0 2 4]", stub.pageCalls)
	}
}

func TestGetCatalogStartsAfterTheVersionOfTheRequest(t *testing.T) {
	t.Parallel()
	_, address := serveStub(t, syncprotocol.Version)

	entries, err := syncclient.NewCatalogClient(address, http.DefaultClient, func() string { return "t" }).GetCatalog(t.Context(), 3)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 || entries[0].GetVersion() != 4 || entries[1].GetVersion() != 5 {
		t.Errorf("entries = %v, want versions 4 and 5", entries)
	}
}

func TestTheWrappersReturnWhatTheServerAnswered(t *testing.T) {
	t.Parallel()
	_, address := serveStub(t, syncprotocol.Version)
	calls := newStubCalls(address, func() string { return "t" })
	vault := bytes.Repeat([]byte{1}, 16)

	appended, err := calls.catalog.AppendCatalogEntry(t.Context(), newCatalogEntry())
	if err != nil || !appended.GetCommitted() || appended.GetHead().GetVersion() != 3 {
		t.Errorf("AppendCatalogEntry() = %v, %v, want a committed answer with the head", appended, err)
	}
	usage, err := calls.catalog.GetVaultUsage(t.Context())
	if err != nil || len(usage) != 1 || usage[0].GetUsedBytes() != 4096 {
		t.Errorf("GetVaultUsage() = %v, %v, want one vault of 4096 bytes", usage, err)
	}
	removed, err := calls.catalog.DeleteVault(t.Context(), vault)
	if err != nil || !removed.GetDeleted() || removed.GetFreedBytes() != 16 {
		t.Errorf("DeleteVault() = %v, %v, want a deleted vault that freed 16 bytes", removed, err)
	}
	committed, err := calls.revision.CommitRevisions(t.Context(), vault, []*syncv1.Revision{newRevision()})
	if err != nil || !committed.GetCommitted() || committed.GetSeq() != 4 {
		t.Errorf("CommitRevisions() = %v, %v, want a committed answer at seq 4", committed, err)
	}
	changes, err := calls.revision.GetChanges(t.Context(), vault, 6, 10)
	if err != nil || changes.GetNextSeq() != 7 || changes.GetHeadSeq() != 9 {
		t.Errorf("GetChanges() = %v, %v, want next 7 and head 9", changes, err)
	}
	uploads, err := calls.blob.ReserveBlobs(t.Context(), vault, []uint64{4096, 8192})
	if err != nil || len(uploads) != 2 || uploads[1].GetSize() != 8192 {
		t.Errorf("ReserveBlobs() = %v, %v, want two uploads in the order of the sizes", uploads, err)
	}
	downloads, err := calls.blob.GetBlobUrls(t.Context(), vault, [][]byte{bytes.Repeat([]byte{4}, 16)})
	if err != nil || len(downloads) != 1 || !downloads[0].GetFound() {
		t.Errorf("GetBlobUrls() = %v, %v, want one download that is found", downloads, err)
	}
}

func TestGetAccountChainByRootReportsWhetherTheAccountExists(t *testing.T) {
	t.Parallel()
	stub, address := serveStub(t, syncprotocol.Version)
	account := syncclient.NewAccountClient(address, http.DefaultClient, func() string { return "" })
	id, root, signature := bytes.Repeat([]byte{1}, 16), bytes.Repeat([]byte{3}, 32), bytes.Repeat([]byte{8}, 64)

	chain, found, err := account.GetAccountChainByRoot(t.Context(), id, root, signature)
	if err != nil || found || len(chain) != 0 {
		t.Errorf("unknown root: chain = %x, found = %v, error = %v, want none", chain, found, err)
	}
	stub.chain = []byte("chain bytes")
	chain, found, err = account.GetAccountChainByRoot(t.Context(), id, root, signature)
	if err != nil || !found || string(chain) != "chain bytes" {
		t.Errorf("known root: chain = %q, found = %v, error = %v, want the chain", chain, found, err)
	}
}

func TestGetAccountChainReturnsTheChainOfTheSession(t *testing.T) {
	t.Parallel()
	stub, address := serveStub(t, syncprotocol.Version)
	stub.chain = []byte("chain bytes")

	chain, err := syncclient.NewAccountClient(address, http.DefaultClient, func() string { return "t" }).GetAccountChain(t.Context())

	if err != nil || string(chain) != "chain bytes" {
		t.Errorf("chain = %q, error = %v, want the chain", chain, err)
	}
}

func TestACallToAServerThatIsNotThereIsUnreachable(t *testing.T) {
	t.Parallel()
	address := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	address.Close()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	_, err := syncclient.NewCatalogClient(address.URL, http.DefaultClient, func() string { return "t" }).GetCatalog(ctx, 0)

	if !errors.Is(err, syncprotocol.ErrUnreachable) {
		t.Errorf("error = %v, want ErrUnreachable", err)
	}
}

type recordingTransport struct {
	mu      sync.Mutex
	headers []http.Header
}

func (r *recordingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.headers = append(r.headers, request.Header.Clone())
	return nil, errors.New("no network in this test")
}

func TestEveryClientSendsNoBearerOverPlainHTTPToAnotherHost(t *testing.T) {
	t.Parallel()
	transport := &recordingTransport{}
	web := &http.Client{Transport: transport}

	if _, err := syncclient.NewCatalogClient("http://sync.example.org", web, func() string { return "token-of-the-session" }).GetCatalog(t.Context(), 4); err == nil {
		t.Error("the catalog call over a transport with no network succeeded, want an error")
	}
	if _, err := syncclient.NewCatalogClient("https://sync.example.org", web, func() string { return "token-of-the-session" }).GetCatalog(t.Context(), 4); err == nil {
		t.Error("the catalog call over a transport with no network succeeded, want an error")
	}

	if len(transport.headers) != 2 {
		t.Fatalf("the transport saw %d requests, want 2", len(transport.headers))
	}
	if got := transport.headers[0].Get("Authorization"); got != "" {
		t.Errorf("plain http sent Authorization = %q, want none", got)
	}
	if got := transport.headers[1].Get("Authorization"); got != "Bearer token-of-the-session" {
		t.Errorf("https sent Authorization = %q, want the bearer", got)
	}
}

func TestDeleteAccountReturnsTheTimeTheServerDeletesTheAccountAndExportAccountTheDocument(t *testing.T) {
	t.Parallel()
	_, address := serveStub(t, syncprotocol.Version)
	account := syncclient.NewAccountClient(address, http.DefaultClient, func() string { return "token" })

	deleteAt, err := account.DeleteAccount(t.Context(), bytes.Repeat([]byte{1}, 16), bytes.Repeat([]byte{8}, 64))
	if err != nil || deleteAt.Unix() != 1_800_000_000 {
		t.Errorf("delete at = %v, error = %v, want 1800000000", deleteAt, err)
	}
	document, err := account.ExportAccount(t.Context())
	if err != nil || string(document) != `{"vaults":[]}` {
		t.Errorf("document = %q, error = %v, want the JSON of the server", document, err)
	}
}

func TestSignInOfADeletingAccountIsAnAccountDeletingErrorWithTheDate(t *testing.T) {
	t.Parallel()
	failure := connect.NewError(connect.CodeUnauthenticated, errors.New("sign-in failed"))
	detail, err := connect.NewErrorDetail(&syncv1.AccountDeleting{DeleteAt: 1_800_000_000})
	if err != nil {
		t.Fatal(err)
	}
	failure.AddDetail(detail)

	got := syncprotocol.ReadError(failure)

	deleting, ok := errors.AsType[syncprotocol.AccountDeletingError](got)
	if !ok || deleting.DeleteAt.Unix() != 1_800_000_000 {
		t.Errorf("error = %v, want AccountDeletingError at 1800000000", got)
	}
}
