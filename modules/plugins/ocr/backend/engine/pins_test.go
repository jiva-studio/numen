package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
)

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

const pinnedTestAddress = "https://example.invalid/pinned-test/model.onnx"

func pinTestModel(t *testing.T, body string) {
	t.Helper()
	digest := sha256.Sum256([]byte(body))
	pinnedSums[pinnedTestAddress] = hex.EncodeToString(digest[:])
	t.Cleanup(func() { delete(pinnedSums, pinnedTestAddress) })
}

func TestEveryDefaultModelIsPinned(t *testing.T) {
	cfg := Defaults()
	for name, address := range map[string]string{
		"layout":     cfg.Layout.Name,
		"detector":   cfg.Detect.Name,
		"recogniser": cfg.Recognise.Name,
	} {
		if !sha256Hex.MatchString(pinnedSums.GetSum(address)) {
			t.Errorf("the %s model %s has no sha256 pinned", name, address)
		}
	}
}

func TestADefaultModelAddressNamesARevision(t *testing.T) {
	for _, address := range []string{layoutDefault, detectDefault, recogDefault} {
		if !regexp.MustCompile(`/resolve/[0-9a-f]{40}/`).MatchString(address) {
			t.Errorf("%s is not at a commit", address)
		}
	}
}

func TestAModelAPersonNamedIsNotPinned(t *testing.T) {
	if got := pinnedSums.GetSum("https://example.org/model.onnx"); got != "" {
		t.Errorf("pinned sum %q for an address nobody pinned", got)
	}
}

func TestIsFetchedFindsAPinnedModelUnderItsSum(t *testing.T) {
	pinTestModel(t, "weights")
	dir := t.TempDir()
	cfg := Config{Dir: dir}
	held := RecogniserModel{Name: pinnedTestAddress}
	if IsFetched(cfg, held) {
		t.Fatal("an empty folder holds the model")
	}
	at := filepath.Join(dir, models.GetCacheName(pinnedTestAddress, pinnedSums.GetSum(pinnedTestAddress)))
	if err := os.WriteFile(at, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsFetched(cfg, held) {
		t.Fatal("the pinned model in the folder is not found")
	}
}

func TestIsFetchedRefusesAPinnedModelWithAnotherSum(t *testing.T) {
	pinTestModel(t, "weights")
	dir := t.TempDir()
	at := filepath.Join(dir, models.GetCacheName(pinnedTestAddress, pinnedSums.GetSum(pinnedTestAddress)))
	if err := os.WriteFile(at, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if IsFetched(Config{Dir: dir}, RecogniserModel{Name: pinnedTestAddress}) {
		t.Fatal("a file with another sum counts as fetched")
	}
}

func TestIsFetchedFindsAModelBesideTheApplicationOnlyWithThePinnedSum(t *testing.T) {
	pinTestModel(t, "weights")
	dir := t.TempDir()
	cfg := Config{Dir: dir}
	held := RecogniserModel{Name: pinnedTestAddress}
	if err := os.WriteFile(filepath.Join(dir, "model.onnx"), []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	if IsFetched(cfg, held) {
		t.Fatal("a file with another sum counts as fetched")
	}
	if err := os.WriteFile(filepath.Join(dir, "model.onnx"), []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsFetched(cfg, held) {
		t.Fatal("the pinned file is not found")
	}
}

func TestIsFetchedOfAPathOrAnUnnamedModel(t *testing.T) {
	file := filepath.Join(t.TempDir(), "rec.onnx")
	if IsFetched(Config{}, RecogniserModel{Path: file}) {
		t.Fatal("a missing path counts as fetched")
	}
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsFetched(Config{}, RecogniserModel{Path: file}) {
		t.Fatal("a present path does not count")
	}
	if IsFetched(Config{}, RecogniserModel{}) {
		t.Fatal("an unnamed model counts as fetched")
	}
	if IsFetched(Config{Dir: t.TempDir()}, RecogniserModel{Name: "rec-model-not-here.onnx"}) {
		t.Fatal("a name that is no address and no file counts as fetched")
	}
}
