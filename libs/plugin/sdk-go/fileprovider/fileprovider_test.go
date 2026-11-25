package fileprovider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

type recorder struct {
	calls []string
	err   error
}

func (r *recorder) FetchFile(_ context.Context, vaultID, path string) error {
	r.calls = append(r.calls, "fetch "+vaultID+" "+path)
	return r.err
}

func (r *recorder) DropDownload(_ context.Context, vaultID, path string) error {
	r.calls = append(r.calls, "drop "+vaultID+" "+path)
	return r.err
}

func (r *recorder) GetFetchProgress(_ context.Context, vaultID, path string) (Progress, error) {
	r.calls = append(r.calls, "progress "+vaultID+" "+path)
	return Progress{IsKnown: true, BytesDone: 4, BytesTotal: 7}, r.err
}

func serve(t *testing.T, h Handler) numenv1connect.FileProviderServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(NewHandler(h))
	web := httptest.NewServer(mux)
	t.Cleanup(web.Close)
	return numenv1connect.NewFileProviderServiceClient(web.Client(), web.URL)
}

func TestTheHandlerPassesTheVaultAndThePathOfEachCallToThePlugin(t *testing.T) {
	plugin := &recorder{}
	client := serve(t, plugin)

	_, fetchErr := client.FetchFile(t.Context(), connect.NewRequest(&v1.FileProviderServiceFetchFileRequest{VaultId: "v-1", Path: "a.pdf"}))
	_, dropErr := client.DropDownload(t.Context(), connect.NewRequest(&v1.FileProviderServiceDropDownloadRequest{VaultId: "v-1", Path: "b.pdf"}))

	if err := errors.Join(fetchErr, dropErr); err != nil {
		t.Fatal(err)
	}
	if len(plugin.calls) != 2 || plugin.calls[0] != "fetch v-1 a.pdf" || plugin.calls[1] != "drop v-1 b.pdf" {
		t.Errorf("the plugin was called with %q", plugin.calls)
	}
}

func TestAFailureOfThePluginReachesTheClientWithItsCodeAndMessage(t *testing.T) {
	plugin := &recorder{err: rpcerror.New(rpcerror.FailedPrecondition, errors.New("The file has changes that are not sent"))}
	client := serve(t, plugin)

	_, err := client.DropDownload(t.Context(), connect.NewRequest(&v1.FileProviderServiceDropDownloadRequest{VaultId: "v-1", Path: "a.pdf"}))

	var failure *connect.Error
	if !errors.As(err, &failure) || failure.Code() != connect.CodeFailedPrecondition || failure.Message() != "The file has changes that are not sent" {
		t.Errorf("the call failed with %v", err)
	}
}

func TestTheHandlerAnswersTheProgressOfThePluginForThePath(t *testing.T) {
	plugin := &recorder{}
	client := serve(t, plugin)

	answer, err := client.GetFetchProgress(t.Context(), connect.NewRequest(&v1.GetFetchProgressRequest{VaultId: "v-1", Path: "a.pdf"}))

	if err != nil {
		t.Fatal(err)
	}
	if !answer.Msg.GetIsKnown() || answer.Msg.GetBytesDone() != 4 || answer.Msg.GetBytesTotal() != 7 {
		t.Errorf("the answer is %v", answer.Msg)
	}
	if len(plugin.calls) != 1 || plugin.calls[0] != "progress v-1 a.pdf" {
		t.Errorf("the plugin was called with %q", plugin.calls)
	}
}
