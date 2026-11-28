package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"slices"
	"strings"
	"testing"
)

// bip39EnglishSHA256 is the SHA-256 of bip-0039/english.txt of the bitcoin/bips repository (2048 lines, each ended by a newline).
const bip39EnglishSHA256 = "2f5eed53a4727b4bf8880d8f3f199efc90e58503646d9ff8eff3a2ed3b24dbda"

func TestWordList_IsTheOfficialBIP39EnglishList(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("english.txt")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if got := hex.EncodeToString(sum[:]); got != bip39EnglishSHA256 {
		t.Errorf("sha256(english.txt) = %s, want %s", got, bip39EnglishSHA256)
	}
	words := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(words) != 2048 {
		t.Fatalf("english.txt has %d words, want 2048", len(words))
	}
	if !slices.IsSorted(words) {
		t.Error("the words are not sorted")
	}
	if len(slices.Compact(slices.Clone(words))) != 2048 {
		t.Error("the words are not unique")
	}
	if words[0] != "abandon" || words[2047] != "zoo" {
		t.Errorf("first and last words are %q and %q", words[0], words[2047])
	}
}
