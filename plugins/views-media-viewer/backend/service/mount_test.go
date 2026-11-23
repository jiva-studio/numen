package service

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/views-media-viewer/backend/api"
)

func readFixture[T any](t *testing.T, kind string) T {
	t.Helper()
	raw, err := os.ReadFile("../../testdata/api/SaveRecording." + kind + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestSaveRecordingCreatesTheCapturedBytesInTheVault(t *testing.T) {
	req := readFixture[api.SaveRequest](t, "request")
	want := readFixture[api.SaveResponse](t, "response")
	vault := hosttest.CreateOnDisk(t, req.VaultID, map[string]string{"Talks/First.md": "# First"})
	saved, err := saveRecording(t.Context(), vault, req)
	if err != nil || saved != want {
		t.Fatalf("save = %+v, %v, want %+v", saved, err, want)
	}
	retried, err := saveRecording(t.Context(), vault, req)
	if err != nil || retried != saved {
		t.Fatalf("retry = %+v, %v, want %+v", retried, err, saved)
	}
	raw, err := os.ReadFile(vault.Vault().Path + "/" + saved.Path)
	if err != nil || !bytes.Equal(raw, req.Content) {
		t.Errorf("stored audio = %q, %v, want %q", raw, err, req.Content)
	}
}
