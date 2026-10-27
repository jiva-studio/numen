package service

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	pdfv1 "github.com/jiva-studio/numen/plugins/readers-pdf/gen/numen/pdf/v1"
	"github.com/jiva-studio/numen/plugins/readers-pdf/gen/numen/pdf/v1/pdfv1connect"
)

// The handler a host that links the plugin in serves answers the document service with the token
// it was given, and refuses a call without it.
func TestTheLinkedHandlerAnswersTheDocumentService(t *testing.T) {
	handler, err := NewHandler(map[string]string{
		"NUMEN_PLUGIN_TOKEN": "linked-token",
		"NUMEN_HOST_PORT":    "1",
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	client := pdfv1connect.NewDocumentServiceClient(web.Client(), web.URL)
	if _, err := client.GetDocument(t.Context(), connect.NewRequest(&pdfv1.GetDocumentRequest{Path: "tiny.pdf"})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("GetDocument without the token = %v, want unauthenticated", err)
	}
	req := connect.NewRequest(&pdfv1.GetDocumentRequest{})
	req.Header().Set("Authorization", "Bearer linked-token")
	if _, err := client.GetDocument(t.Context(), req); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("GetDocument with no path = %v, want invalid argument", err)
	}
}

// A linked handler reaches the host at the port of its variables.
func TestTheLinkedHandlerReachesTheHostOfItsVariables(t *testing.T) {
	hostServer := httptest.NewServer(http.NotFoundHandler())
	defer hostServer.Close()
	port := hostServer.URL[strings.LastIndex(hostServer.URL, ":")+1:]

	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token", "NUMEN_HOST_PORT": port})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	req := connect.NewRequest(&pdfv1.GetDocumentRequest{Path: filepath.Join("books", "tiny.pdf")})
	req.Header().Set("Authorization", "Bearer linked-token")
	_, err = pdfv1connect.NewDocumentServiceClient(web.Client(), web.URL).GetDocument(t.Context(), req)

	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("GetDocument = %v, want not found", err)
	}
}
