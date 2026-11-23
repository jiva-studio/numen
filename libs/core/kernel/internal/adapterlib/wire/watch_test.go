package wire

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

func openLicenses(
	t *testing.T, keepalive func(*v1.WatchLicensesResponse) *v1.WatchLicensesResponse,
	run func(context.Context, func(*v1.WatchLicensesResponse) error) error,
) *connect.ServerStreamForClient[v1.WatchLicensesResponse] {
	t.Helper()
	handler := connect.NewServerStreamHandler(
		numenv1connect.LicenseServiceWatchLicensesProcedure,
		func(ctx context.Context, _ *connect.Request[v1.WatchLicensesRequest], out *connect.ServerStream[v1.WatchLicensesResponse]) error {
			return RunWatch(ctx, out, time.Millisecond, keepalive, run)
		},
	)
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.LicenseServiceWatchLicensesProcedure, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	client := numenv1connect.NewLicenseServiceClient(server.Client(), server.URL)
	stream, err := client.WatchLicenses(t.Context(), connect.NewRequest(&v1.WatchLicensesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stream.Close() })
	return stream
}

func TestAQuietWatchSendsAnEmptyMessageAsKeepalive(t *testing.T) {
	stream := openLicenses(t, nil, func(ctx context.Context, _ func(*v1.WatchLicensesResponse) error) error {
		<-ctx.Done()
		return nil
	})
	if !stream.Receive() {
		t.Fatalf("no keepalive arrived: %v", stream.Err())
	}
	if got := len(stream.Msg().GetLicenses()); got != 0 {
		t.Fatalf("keepalive holds %d licenses, want none", got)
	}
}

func TestAQuietWatchOfTheWholeStateSendsTheStateAgain(t *testing.T) {
	stream := openLicenses(t, GetLastOrEmpty, func(ctx context.Context, send func(*v1.WatchLicensesResponse) error) error {
		if err := send(&v1.WatchLicensesResponse{Licenses: []*v1.LicenseInfo{{Id: "harbor"}}}); err != nil {
			return err
		}
		<-ctx.Done()
		return nil
	})
	for range 3 {
		if !stream.Receive() {
			t.Fatalf("the stream ended: %v", stream.Err())
		}
		if got := stream.Msg().GetLicenses(); len(got) != 1 || got[0].GetId() != "harbor" {
			t.Fatalf("licenses = %v, want the one sent", got)
		}
	}
}
