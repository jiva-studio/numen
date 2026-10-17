package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
)

func TestLocateRefusesToFetchWhileOffline(t *testing.T) {
	t.Setenv(models.EnvOffline, "1")
	t.Setenv(models.EnvDir, t.TempDir())
	_, err := Locate(context.Background(), LocalModel{Name: "owner/repo", ShouldDownload: true}, nil)
	if err == nil || !strings.Contains(err.Error(), models.EnvOffline) {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestLocateRefusesToFetchWhenDownloadIsOff(t *testing.T) {
	_, err := Locate(context.Background(), LocalModel{Name: "owner/repo"}, nil)
	if err == nil || !strings.Contains(err.Error(), "enable download") {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestCheckPinned(t *testing.T) {
	file := filepath.Join(t.TempDir(), "model.onnx")
	if err := os.WriteFile(file, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("weights"))
	sum := hex.EncodeToString(digest[:])

	if err := checkPinned(map[string]string{"onnx/model.onnx": sum}, "onnx/model.onnx", file); err != nil {
		t.Fatalf("matching sum: %v", err)
	}
	err := checkPinned(map[string]string{"onnx/model.onnx": strings.Repeat("0", 64)}, "onnx/model.onnx", file)
	if err == nil || !strings.Contains(err.Error(), "onnx/model.onnx") {
		t.Fatalf("another sum: %v", err)
	}
	if err := checkPinned(map[string]string{"other": sum}, "onnx/model.onnx", file); err != nil {
		t.Fatalf("an unpinned name must not be checked: %v", err)
	}
	if err := checkPinned(nil, "onnx/model.onnx", file); err != nil {
		t.Fatalf("no pins: %v", err)
	}
}
