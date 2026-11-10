package host_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// rangeReading is a host that returns a ranged ReadFile with a fixed range of Glaciers.md, or with
// code when it is set, and records the request.
type rangeReading struct {
	numenv1connect.UnimplementedHostVaultServiceHandler
	code    v1.HostVaultErrorCode
	request *v1.ReadFileRequest
}

func (r *rangeReading) ReadFile(_ context.Context, req *connect.Request[v1.ReadFileRequest]) (*connect.Response[v1.ReadFileResponse], error) {
	r.request = req.Msg
	if r.code != 0 {
		return connect.NewResponse(&v1.ReadFileResponse{Error: r.code}), nil
	}
	return connect.NewResponse(&v1.ReadFileResponse{
		Content: []byte("ice moves slowly"), Start: 12,
		Stat: &v1.HostFileStat{Path: "Glaciers.md", Size: 5000, ModTime: 7000, IsNote: true},
	}), nil
}

func rangeClient(t *testing.T, r *rangeReading) *host.Client {
	t.Helper()
	server := httptest.NewServer(handler(r))
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestTheClientReadsARangeOfAFileWithItsStartAndStat(t *testing.T) {
	r := &rangeReading{}

	got, err := rangeClient(t, r).ReadRange(t.Context(), "Glaciers.md", 10, 300)
	if err != nil {
		t.Fatal(err)
	}

	if r.request.GetPath() != "Glaciers.md" || r.request.GetStart() != 10 || r.request.Length == nil || r.request.GetLength() != 300 {
		t.Errorf("the host received %v, want Glaciers.md from 10 with the length 300 set", r.request)
	}
	if got.Text != "ice moves slowly" || got.Start != 12 {
		t.Errorf("the range came back as %+v, want the text from 12", got)
	}
	if got.Stat.Path != "Glaciers.md" || got.Stat.Size != 5000 || !got.Stat.ModTime.Equal(time.Unix(0, 7000)) {
		t.Errorf("the stat came back as %+v, want Glaciers.md of 5000 bytes at 7000", got.Stat)
	}
}

func TestTheClientSendsALengthOfZeroAsSet(t *testing.T) {
	r := &rangeReading{}

	if _, err := rangeClient(t, r).ReadRange(t.Context(), "Glaciers.md", 0, 0); err != nil {
		t.Fatal(err)
	}

	if r.request.Length == nil {
		t.Errorf("the host received %v with no length, want the length 0 set so the read is ranged", r.request)
	}
}

func TestClientReturnsTheErrorOfARejectedRange(t *testing.T) {
	for code, want := range map[v1.HostVaultErrorCode]error{
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_TEXT:  host.ErrNotText,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND: host.ErrNotExist,
		v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NO_VAULT:  host.ErrNoVault,
	} {
		_, err := rangeClient(t, &rangeReading{code: code}).ReadRange(t.Context(), "Moraine.bin", 0, 100)
		if !errors.Is(err, want) {
			t.Errorf("error %s: got %v, want %v", code, err, want)
		}
	}
}

// rangeTexts is a host of texts that returns a ranged ReadText with two segments cut to the range,
// and records the request.
type rangeTexts struct {
	numenv1connect.UnimplementedHostTextServiceHandler
	request *v1.ReadTextRequest
}

func (h *rangeTexts) ReadText(_ context.Context, req *connect.Request[v1.ReadTextRequest]) (*connect.Response[v1.ReadTextResponse], error) {
	h.request = req.Msg
	return connect.NewResponse(&v1.ReadTextResponse{
		Segments: []*v1.TextSegment{{Text: "lacier ", Selector: `{"page":3}`}, {Text: "calves", Heading: "Ice"}},
		Start:    41, Size: 2400,
	}), nil
}

func TestTheTextClientReadsARangeOfAVersion(t *testing.T) {
	h := &rangeTexts{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostTextServiceHandler(h))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := host.NewTextClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})

	got, err := client.ReadTextRange(t.Context(), "Glaciers.pdf", "org.example.ocr/k", 41, 13)
	if err != nil {
		t.Fatal(err)
	}

	r := h.request
	if r.GetPath() != "Glaciers.pdf" || r.GetVersionId() != "org.example.ocr/k" || r.GetStart() != 41 || r.Length == nil || r.GetLength() != 13 {
		t.Errorf("the host received %v, want Glaciers.pdf, version org.example.ocr/k, from 41 with the length 13 set", r)
	}
	want := []domain.TextSegment{{Text: "lacier ", Selector: `{"page":3}`}, {Text: "calves", Heading: "Ice"}}
	if !slices.Equal(got.Segments, want) || got.Start != 41 || got.Size != 2400 {
		t.Errorf("the range came back as %+v, want %+v from 41 of 2400 bytes", got, want)
	}
}
