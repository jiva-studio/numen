package client_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

type restoreStub struct {
	syncv1connect.UnimplementedRestoreServiceHandler

	mu       sync.Mutex
	headers  []http.Header
	response *syncv1.GetRestoreEpochResponse
}

func (s *restoreStub) GetRestoreEpoch(_ context.Context, req *connect.Request[syncv1.GetRestoreEpochRequest]) (*connect.Response[syncv1.GetRestoreEpochResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.headers = append(s.headers, req.Header().Clone())
	return connect.NewResponse(s.response), nil
}

func serveRestoreStub(tb testing.TB, response *syncv1.GetRestoreEpochResponse) (*restoreStub, *syncclient.RestoreClient) {
	tb.Helper()
	stub := &restoreStub{response: response}
	mux := http.NewServeMux()
	mux.Handle(syncv1connect.NewRestoreServiceHandler(stub, connect.WithInterceptors(syncprotocol.NewVersionHandlerInterceptor(syncprotocol.Version))))
	server := httptest.NewServer(mux)
	tb.Cleanup(server.Close)
	return stub, syncclient.NewRestoreClient(server.URL, http.DefaultClient, func() string { return "token-of-the-session" })
}

func newEpochOf(tb testing.TB, seed byte) (*syncv1.GetRestoreEpochResponse, ed25519.PrivateKey) {
	tb.Helper()
	private := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, ed25519.SeedSize))
	public, ok := private.Public().(ed25519.PublicKey)
	if !ok {
		tb.Fatal("not an Ed25519 key")
	}
	id := bytes.Repeat([]byte{0x31}, 16)
	epoch := &syncv1.RestoreEpoch{Id: id, IssuedAt: 1_800_000_000, Signature: ed25519.Sign(private, syncprotocol.GetRestoreEpochMessage(id, 1_800_000_000))}
	return &syncv1.GetRestoreEpochResponse{EpochSignKey: public, Epoch: epoch}, private
}

func TestRestoreClient_ReturnsTheEpochAndTheKeyWithTheBearerOfTheToken(t *testing.T) {
	t.Parallel()
	response, _ := newEpochOf(t, 1)
	stub, client := serveRestoreStub(t, response)

	epoch, key, err := client.GetRestoreEpoch(t.Context())

	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(epoch.GetId(), response.GetEpoch().GetId()) || !bytes.Equal(key, response.GetEpochSignKey()) {
		t.Errorf("epoch %x key %x, want what the server sent", epoch.GetId(), key)
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.headers) != 1 || stub.headers[0].Get("Authorization") != "Bearer token-of-the-session" {
		t.Errorf("headers = %v, want one request with the bearer", stub.headers)
	}
}

func TestRestoreClient_ReturnsNoEpochBeforeTheFirstRestore(t *testing.T) {
	t.Parallel()
	response, _ := newEpochOf(t, 1)
	response.Epoch = nil
	_, client := serveRestoreStub(t, response)

	epoch, key, err := client.GetRestoreEpoch(t.Context())

	if err != nil {
		t.Fatal(err)
	}
	if epoch != nil || !bytes.Equal(key, response.GetEpochSignKey()) {
		t.Errorf("epoch %v key %x, want no epoch and the key", epoch, key)
	}
}

func TestRestoreClient_RefusesAnEpochWhoseSignatureDoesNotVerify(t *testing.T) {
	t.Parallel()
	response, _ := newEpochOf(t, 1)
	response.Epoch.IssuedAt++
	_, client := serveRestoreStub(t, response)

	_, _, err := client.GetRestoreEpoch(t.Context())

	if !errors.Is(err, syncprotocol.ErrInvalidRestoreEpoch) {
		t.Errorf("GetRestoreEpoch = %v, want ErrInvalidRestoreEpoch", err)
	}
}

func TestRestoreClient_RefusesAKeyThatIsNotThirtyTwoBytes(t *testing.T) {
	t.Parallel()
	response, _ := newEpochOf(t, 1)
	response.EpochSignKey = response.EpochSignKey[:31]
	_, client := serveRestoreStub(t, response)

	_, _, err := client.GetRestoreEpoch(t.Context())

	if !errors.Is(err, syncprotocol.ErrInvalidRestoreEpoch) {
		t.Errorf("GetRestoreEpoch = %v, want ErrInvalidRestoreEpoch", err)
	}
}

type blobStub struct {
	syncv1connect.UnimplementedBlobServiceHandler

	fail error
}

func (s *blobStub) GetBlobUrls(context.Context, *connect.Request[syncv1.GetBlobUrlsRequest]) (*connect.Response[syncv1.GetBlobUrlsResponse], error) {
	return nil, s.fail
}

func newBlobMissingFailure(tb testing.TB, ids ...[]byte) error {
	tb.Helper()
	failure := connect.NewError(connect.CodeNotFound, errors.New("the body is gone"))
	detail, err := connect.NewErrorDetail(&syncv1.BlobMissing{BlobIds: ids})
	if err != nil {
		tb.Fatal(err)
	}
	failure.AddDetail(detail)
	return failure
}

func newBlobClientFailing(tb testing.TB, failure error) *syncclient.BlobClient {
	tb.Helper()
	mux := http.NewServeMux()
	mux.Handle(syncv1connect.NewBlobServiceHandler(&blobStub{fail: failure}, connect.WithInterceptors(syncprotocol.NewVersionHandlerInterceptor(syncprotocol.Version))))
	server := httptest.NewServer(mux)
	tb.Cleanup(server.Close)
	return syncclient.NewBlobClient(server.URL, http.DefaultClient, func() string { return "token-of-the-session" })
}

func TestBlobClient_MapsNotFoundWithTheMissingBlobDetailToBlobMissingError(t *testing.T) {
	t.Parallel()
	blobID := bytes.Repeat([]byte{4}, 16)
	client := newBlobClientFailing(t, newBlobMissingFailure(t, blobID))

	_, err := client.GetBlobUrls(t.Context(), bytes.Repeat([]byte{1}, 16), [][]byte{blobID})

	missing, ok := errors.AsType[syncprotocol.BlobMissingError](err)
	if !ok || len(missing.IDs) != 1 || !bytes.Equal(missing.IDs[0], blobID) {
		t.Fatalf("GetBlobUrls = %v, want BlobMissingError naming the blob", err)
	}
	if !errors.Is(err, syncprotocol.ErrBlobMissing) {
		t.Error("the error does not read as ErrBlobMissing")
	}
	if errors.Is(err, syncprotocol.ErrNoVault) {
		t.Error("a missing blob reads as a missing vault")
	}
}

func TestBlobClient_KeepsNotFoundWithoutADetailAsNoVault(t *testing.T) {
	t.Parallel()
	client := newBlobClientFailing(t, connect.NewError(connect.CodeNotFound, errors.New("no such vault")))

	_, err := client.GetBlobUrls(t.Context(), bytes.Repeat([]byte{1}, 16), [][]byte{bytes.Repeat([]byte{4}, 16)})

	if !errors.Is(err, syncprotocol.ErrNoVault) || errors.Is(err, syncprotocol.ErrBlobMissing) {
		t.Errorf("GetBlobUrls = %v, want ErrNoVault only", err)
	}
}
