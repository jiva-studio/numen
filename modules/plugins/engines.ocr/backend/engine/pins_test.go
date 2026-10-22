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

func TestIsFetchedFindsAPinnedModelBelowItsSum(t *testing.T) {
	pinTestModel(t, "weights")
	dir := t.TempDir()
	cfg := Config{Dir: dir}
	target := RecogniserModel{Name: pinnedTestAddress}
	if IsFetched(cfg, target) {
		t.Fatal("an empty folder holds the model")
	}
	pos := filepath.Join(dir, models.GetCacheName(pinnedTestAddress, pinnedSums.GetSum(pinnedTestAddress)))
	if err := os.WriteFile(pos, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsFetched(cfg, target) {
		t.Fatal("the pinned model in the folder is not found")
	}
}

func TestIsFetchedRefusesAPinnedModelWithAnotherSum(t *testing.T) {
	pinTestModel(t, "weights")
	dir := t.TempDir()
	pos := filepath.Join(dir, models.GetCacheName(pinnedTestAddress, pinnedSums.GetSum(pinnedTestAddress)))
	if err := os.WriteFile(pos, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if IsFetched(Config{Dir: dir}, RecogniserModel{Name: pinnedTestAddress}) {
		t.Fatal("a file with another sum counts as fetched")
	}
}

func TestIsFetchedFindsAModelAdjacentTheApplicationOnlyWithThePinnedSum(t *testing.T) {
	pinTestModel(t, "weights")
	dir := t.TempDir()
	cfg := Config{Dir: dir}
	target := RecogniserModel{Name: pinnedTestAddress}
	if err := os.WriteFile(filepath.Join(dir, "model.onnx"), []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	if IsFetched(cfg, target) {
		t.Fatal("a file with another sum counts as fetched")
	}
	if err := os.WriteFile(filepath.Join(dir, "model.onnx"), []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsFetched(cfg, target) {
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

func TestIsFetchedUnpinnedModel(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Dir: dir}
	address := "https://example.invalid/unpinned/model.onnx"
	target := RecogniserModel{Name: address}
	if IsFetched(cfg, target) {
		t.Fatal("an unpinned model not in cache counts as fetched")
	}
	cacheFile := filepath.Join(dir, models.GetCacheName(address, ""))
	if err := os.WriteFile(cacheFile, []byte("unpinned-weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsFetched(cfg, target) {
		t.Fatal("an unpinned model in cache is not fetched")
	}
}

func TestIsFetchedNonAddressIgnoredEvenIfCacheFileExists(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Dir: dir}
	name := "custom-model.onnx"
	target := RecogniserModel{Name: name}
	cacheFile := filepath.Join(dir, models.GetCacheName(name, ""))
	if err := os.WriteFile(cacheFile, []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	if IsFetched(cfg, target) {
		t.Fatal("a non-address name found via cache counts as fetched")
	}
}

func TestIsFetchedReturnsFalseWhenGetDirFails(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	address := "https://example.invalid/unpinned/model.onnx"
	cacheName := models.GetCacheName(address, "")
	if err := os.WriteFile(filepath.Join(dir, cacheName), []byte("weights"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NUMEN_MODELS_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("APPDATA", "")
	t.Setenv("LOCALAPPDATA", "")

	cfg := Config{}
	if _, err := cfg.folder().GetDir(); err == nil {
		t.Fatal("cfg.folder().GetDir() succeeded without environment directory variables")
	}
	target := RecogniserModel{Name: address}
	if IsFetched(cfg, target) {
		t.Fatal("IsFetched returned true when folder().GetDir() failed")
	}
}
