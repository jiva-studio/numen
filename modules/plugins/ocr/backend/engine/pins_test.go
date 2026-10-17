package engine

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
)

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

func TestEveryDefaultModelIsPinned(t *testing.T) {
	cfg := Defaults()
	for name, address := range map[string]string{
		"layout":     cfg.Layout.Name,
		"detector":   cfg.Detect.Name,
		"recogniser": cfg.Recognise.Name,
	} {
		if !sha256Hex.MatchString(getPinnedSum(address)) {
			t.Errorf("the %s model %s has no sha256 pinned", name, address)
		}
	}
}

func TestADefaultModelAddressNamesARevision(t *testing.T) {
	for _, address := range []string{layoutDefault, detectDefault} {
		if !regexp.MustCompile(`/resolve/[0-9a-f]{40}/`).MatchString(address) {
			t.Errorf("%s is not at a commit", address)
		}
	}
}

func TestAModelAPersonNamedIsNotPinned(t *testing.T) {
	if got := getPinnedSum("https://example.org/model.onnx"); got != "" {
		t.Errorf("pinned sum %q for an address nobody pinned", got)
	}
}

func TestIsFetchedFindsAPinnedModelUnderItsSum(t *testing.T) {
	dir := t.TempDir()
	cfg := Defaults()
	cfg.Dir = dir
	if IsFetched(cfg, cfg.Recognise) {
		t.Fatal("an empty folder holds the model")
	}
	at := filepath.Join(dir, models.GetCacheName(recogDefault, getPinnedSum(recogDefault)))
	if err := os.WriteFile(at, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsFetched(cfg, cfg.Recognise) {
		t.Fatal("the pinned model in the folder is not found")
	}
}
