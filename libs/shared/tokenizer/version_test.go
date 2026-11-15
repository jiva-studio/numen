package tokenizer_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/shared/tokenizer"
)

func TestGetVersionOfAUnigramTokenizerIsThePackageVersion(t *testing.T) {
	if got := loadFixture(t).GetVersion(); got != tokenizer.Version || got == "" {
		t.Errorf("GetVersion() = %q, want %q", got, tokenizer.Version)
	}
}

func TestGetVersionOfADelegatedTokenizerIsEmpty(t *testing.T) {
	if got := loadTokenizer(t, bpePath).GetVersion(); got != "" {
		t.Errorf("GetVersion() = %q, want an empty version", got)
	}
}
