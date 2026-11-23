package slip39_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"slices"
	"strings"
	"testing"
)

// wordListSHA256 is the SHA-256 of shamir_mnemonic/wordlist.txt of trezor/python-shamir-mnemonic (1024 lines, each ended by a newline).
const wordListSHA256 = "bcc4555340332d169718aed8bf31dd9d5248cb7da6e5d355140ef4f1e601eec3"

func readWordList(tb testing.TB) []string {
	tb.Helper()
	raw, err := os.ReadFile("wordlist.txt")
	if err != nil {
		tb.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
}

func TestWordList_IsTheOfficialSLIP39List(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("wordlist.txt")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if got := hex.EncodeToString(sum[:]); got != wordListSHA256 {
		t.Errorf("sha256(wordlist.txt) = %s, want %s", got, wordListSHA256)
	}
	words := readWordList(t)
	if len(words) != 1024 {
		t.Fatalf("wordlist.txt has %d words, want 1024", len(words))
	}
	if !slices.IsSorted(words) {
		t.Error("the words are not sorted")
	}
	if len(slices.Compact(slices.Clone(words))) != 1024 {
		t.Error("the words are not unique")
	}
	if words[0] != "academic" || words[1023] != "zero" {
		t.Errorf("first and last words are %q and %q", words[0], words[1023])
	}
}

func TestWordList_HasUniqueFourLetterPrefixes(t *testing.T) {
	t.Parallel()
	seen := make(map[string]string)
	for _, word := range readWordList(t) {
		prefix := word[:min(4, len(word))]
		if other, ok := seen[prefix]; ok {
			t.Errorf("%q and %q share the prefix %q", other, word, prefix)
		}
		seen[prefix] = word
	}
}

func TestNotice_NamesTheLicenseOfTheWordListAndTheVectors(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("NOTICE")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"SatoshiLabs", "MIT"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("NOTICE does not name %q", want)
		}
	}
}
