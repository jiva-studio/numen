package service

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

type askedHost struct {
	numenv1connect.UnimplementedHostServiceHandler
	purposes []string
	auth     string
}

func (h *askedHost) Complete(_ context.Context, req *connect.Request[numenv1.HostServiceCompleteRequest]) (*connect.Response[numenv1.HostServiceCompleteResponse], error) {
	h.purposes = append(h.purposes, req.Msg.GetQuestion().GetPurpose())
	h.auth = req.Header().Get("Authorization")
	return connect.NewResponse(&numenv1.HostServiceCompleteResponse{Answer: &numenv1.CompletionAnswer{Text: "1: words"}}), nil
}

func proofreadWords(t *testing.T, vars map[string]string) error {
	t.Helper()
	handler, err := NewHandler(vars)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	client := numenv1connect.NewProofreadingServiceClient(web.Client(), web.URL)
	req := connect.NewRequest(speechRequest(&numenv1.ProofreadSpeechOptions{Size: 4}, line(1, "words")))
	req.Header().Set("Authorization", "Bearer linked-token")
	_, err = client.ProofreadText(t.Context(), req)
	return err
}

// A linked handler asks the host its variables name, with its own token.
func TestTheLinkedHandlerAsksTheHostOfItsVariables(t *testing.T) {
	request := &askedHost{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostServiceHandler(request))
	host := httptest.NewServer(mux)
	defer host.Close()

	err := proofreadWords(t, map[string]string{
		"NUMEN_PLUGIN_TOKEN": "linked-token",
		"NUMEN_HOST_PORT":    strconv.Itoa(host.Listener.Addr().(*net.TCPAddr).Port),
	})

	if err != nil {
		t.Fatalf("ProofreadText: %v", err)
	}
	if len(request.purposes) == 0 || request.purposes[0] != purpose || request.auth != "Bearer linked-token" {
		t.Fatalf("the host was asked %v with %q", request.purposes, request.auth)
	}
}

// A linked handler whose variables name no host answers unavailable.
func TestTheLinkedHandlerWithoutAHostIsUnavailable(t *testing.T) {
	err := proofreadWords(t, map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})

	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("ProofreadText = %v, want unavailable", err)
	}
}
