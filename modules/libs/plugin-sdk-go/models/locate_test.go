package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pinnedTestAddress = "https://example.invalid/pinned-test/model.onnx"

func pinTestModel(body string) Pins {
	digest := sha256.Sum256([]byte(body))
	return Pins{pinnedTestAddress: hex.EncodeToString(digest[:])}
}

func sourceIn(dir string) Source {
	return Source{
		GetPaths: func(file string) []string { return []string{filepath.Join(dir, file)} },
		Fetch: func(context.Context, string, string) (string, error) {
			return "", errors.New("fetch refused")
		},
	}
}

func TestAModelBesideTheApplicationIsUsedWhenItHoldsThePinnedSum(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "model.onnx")
	if err := os.WriteFile(at, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Locate(context.Background(), sourceIn(dir), pinTestModel("weights"), "", pinnedTestAddress, "layout model")
	if err != nil || got != at {
		t.Fatalf("Locate() = %q, %v", got, err)
	}
}

func TestAModelBesideTheApplicationWithAnotherSumIsNotUsed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "model.onnx"), []byte("another"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Locate(context.Background(), sourceIn(dir), pinTestModel("weights"), "", pinnedTestAddress, "layout model")
	if err == nil || !strings.Contains(err.Error(), "fetch refused") {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestAModelNobodyPinnedBesideTheApplicationIsUsedAsFound(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "model.onnx")
	if err := os.WriteFile(at, []byte("anything"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Locate(context.Background(), sourceIn(dir), nil, "", "https://example.invalid/unpinned/model.onnx", "layout model")
	if err != nil || got != at {
		t.Fatalf("Locate() = %q, %v", got, err)
	}
}

func TestAFetchedModelIsAskedForUnderItsPinnedSum(t *testing.T) {
	pins := pinTestModel("weights")
	src := sourceIn(t.TempDir())
	var gotSum string
	src.Fetch = func(_ context.Context, _, sum string) (string, error) {
		gotSum = sum
		return "fetched.onnx", nil
	}

	got, err := Locate(context.Background(), src, pins, "", pinnedTestAddress, "layout model")
	if err != nil || got != "fetched.onnx" || gotSum != pins[pinnedTestAddress] {
		t.Fatalf("Locate() = %q, %v, sum %q", got, err, gotSum)
	}
}

func TestAPathThatIsMissingIsAnError(t *testing.T) {
	_, err := Locate(context.Background(), sourceIn(t.TempDir()), nil, filepath.Join(t.TempDir(), "none.onnx"), "", "layout model")
	if err == nil || !strings.Contains(err.Error(), "the layout model") {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestAModelWithNoNameAndNoPathIsAnError(t *testing.T) {
	_, err := Locate(context.Background(), sourceIn(t.TempDir()), nil, "", "", "encoder")
	if err == nil || !strings.Contains(err.Error(), "no encoder") {
		t.Fatalf("Locate() error = %v", err)
	}
}

func TestANameThatIsNoAddressAndNoFileIsAnError(t *testing.T) {
	_, err := Locate(context.Background(), sourceIn(t.TempDir()), nil, "", "missing.onnx", "encoder")
	if err == nil || !strings.Contains(err.Error(), "not beside the application") {
		t.Fatalf("Locate() error = %v", err)
	}
}
