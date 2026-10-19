package service

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

func openRecording(t *testing.T, vault, path string) error {
	t.Helper()
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token", "NUMEN_VAULT_PATH": vault})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()
	req := connect.NewRequest(&numenv1.OpenRecordingRequest{Path: path})
	req.Header().Set("Authorization", "Bearer linked-token")
	_, err = numenv1connect.NewTranscriptionServiceClient(web.Client(), web.URL).OpenRecording(t.Context(), req)
	return err
}

// A linked handler reads audio from the vault of its variables alone.
func TestTheLinkedHandlerRefusesAudioOutsideTheVaultOfItsVariables(t *testing.T) {
	vault := t.TempDir()

	outside := openRecording(t, vault, filepath.Join(t.TempDir(), "talk.wav"))
	inside := openRecording(t, vault, "talk.wav")

	if connect.CodeOf(outside) != connect.CodePermissionDenied {
		t.Errorf("a path outside the vault = %v, want permission denied", outside)
	}
	if connect.CodeOf(inside) == connect.CodePermissionDenied {
		t.Errorf("a path inside the vault = %v, want it let through to the read", inside)
	}
}
