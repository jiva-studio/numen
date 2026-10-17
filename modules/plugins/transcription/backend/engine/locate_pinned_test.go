package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pinnedTestAddress = "https://example.invalid/pinned-test/model.onnx"

func pinTestModel(t *testing.T, body string) {
	t.Helper()
	digest := sha256.Sum256([]byte(body))
	pinnedSums[pinnedTestAddress] = hex.EncodeToString(digest[:])
	t.Cleanup(func() { delete(pinnedSums, pinnedTestAddress) })
}

func TestAModelBesideTheApplicationIsUsedWhenItHoldsThePinnedSum(t *testing.T) {
	pinTestModel(t, "weights")
	dir := t.TempDir()
	at := filepath.Join(dir, "model.onnx")
	if err := os.WriteFile(at, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Dir: dir}
	address := pinnedTestAddress

	got, err := model(context.Background(), cfg, "", address, "segmenter model")
	if err != nil || got != at {
		t.Fatalf("model() = %q, %v", got, err)
	}
}

func TestAModelBesideTheApplicationWithAnotherSumIsNotUsed(t *testing.T) {
	pinTestModel(t, "weights")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "model.onnx"), []byte("another"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Dir: dir}
	address := pinnedTestAddress

	_, err := model(context.Background(), cfg, "", address, "segmenter model")
	if err == nil || !strings.Contains(err.Error(), ".download") {
		t.Fatalf("model() error = %v", err)
	}
}

func TestAModelNobodyPinnedBesideTheApplicationIsUsedAsFound(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "model.onnx")
	if err := os.WriteFile(at, []byte("anything"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Dir: dir}
	address := "https://example.invalid/unpinned/model.onnx"

	got, err := model(context.Background(), cfg, "", address, "segmenter model")
	if err != nil || got != at {
		t.Fatalf("model() = %q, %v", got, err)
	}
}
