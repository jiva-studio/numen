package engine

import (
	"regexp"
	"strings"
	"testing"
)

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

func TestEveryDefaultModelFileIsPinned(t *testing.T) {
	cfg := Defaults()
	base := strings.TrimSuffix(cfg.Model.Repo, "/") + "/"
	for _, address := range []string{
		base + encoderFile, base + decoderFile, base + joinerFile, base + tokensFile, cfg.Segmenter.Repo,
	} {
		if !sha256Hex.MatchString(pinnedSums.GetSum(address)) {
			t.Errorf("%s has no sha256 pinned", address)
		}
	}
}

func TestTheDefaultAddressesNameARevision(t *testing.T) {
	for _, address := range []string{parakeetDefault, segmenterDefault} {
		if !regexp.MustCompile(`/resolve/[0-9a-f]{40}/`).MatchString(address) {
			t.Errorf("%s is not at a commit", address)
		}
	}
}

func TestAModelAPersonNamedIsNotPinned(t *testing.T) {
	if got := pinnedSums.GetSum("https://example.org/encoder.int8.onnx"); got != "" {
		t.Errorf("pinned sum %q for an address nobody pinned", got)
	}
}
