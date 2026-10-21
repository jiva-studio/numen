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
			return "", errors.New("fetch failed")
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
	if err == nil || !strings.Contains(err.Error(), "fetch failed") {
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

func TestLocateAllFillsDestinations(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "m1.onnx")
	p2 := filepath.Join(dir, "m2.onnx")
	if err := os.WriteFile(p1, []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p2, []byte("2"), 0o644); err != nil {
		t.Fatal(err)
	}
	var d1, d2 string
	err := LocateAll(context.Background(), sourceIn(dir), nil, []Wanted{
		{Dst: &d1, Path: p1, Kind: "first"},
		{Dst: &d2, Path: p2, Kind: "second"},
	})
	if err != nil {
		t.Fatalf("LocateAll() error = %v", err)
	}
	if d1 != p1 || d2 != p2 {
		t.Errorf("got d1=%q, d2=%q, want %q and %q", d1, d2, p1, p2)
	}
}

func TestLocateAllFailsFastOnError(t *testing.T) {
	var d1, d2 string
	err := LocateAll(context.Background(), sourceIn(t.TempDir()), nil, []Wanted{
		{Dst: &d1, Path: "/nonexistent/1.onnx", Kind: "first"},
		{Dst: &d2, Path: "/nonexistent/2.onnx", Kind: "second"},
	})
	if err == nil {
		t.Fatal("LocateAll() expected error on missing path")
	}
	if !strings.Contains(err.Error(), "the first") {
		t.Errorf("expected first error, got %v", err)
	}
	if d1 != "" || d2 != "" {
		t.Errorf("destinations were modified on error: d1=%q, d2=%q", d1, d2)
	}
}

