package service

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// The handler a host that links the plugin in serves answers the document service with the token
// it was given.
func TestTheLinkedHandlerAnswersTheDocumentService(t *testing.T) {
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token"})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	path, err := filepath.Abs(filepath.Join("..", "pdf", "testdata", "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	client := numenv1connect.NewDocumentServiceClient(web.Client(), web.URL)
	req := connect.NewRequest(&numenv1.GetDocumentRequest{Path: path})
	req.Header().Set("Authorization", "Bearer linked-token")
	resp, err := client.GetDocument(t.Context(), req)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if len(resp.Msg.GetPages()) == 0 {
		t.Fatal("the document has no pages")
	}
}

// A linked handler holds the paths of requests to the vault of its variables.
func TestTheLinkedHandlerRefusesADocumentOutsideTheVaultOfItsVariables(t *testing.T) {
	t.Setenv("NUMEN_VAULT_PATH", "")
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token", "NUMEN_VAULT_PATH": t.TempDir()})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	path, err := filepath.Abs(filepath.Join("..", "pdf", "testdata", "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}

	req := connect.NewRequest(&numenv1.GetDocumentRequest{Path: path})
	req.Header().Set("Authorization", "Bearer linked-token")
	_, err = numenv1connect.NewDocumentServiceClient(web.Client(), web.URL).GetDocument(t.Context(), req)

	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("GetDocument outside the vault = %v, want permission denied", err)
	}
}
