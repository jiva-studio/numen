package identity_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/iotest"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func bip39Vectors() []struct{ entropy, words string } {
	return []struct{ entropy, words string }{
		{"00000000000000000000000000000000", "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"},
		{"7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f7f", "legal winner thank year wave sausage worth useful legal winner thank yellow"},
		{"80808080808080808080808080808080", "letter advice cage absurd amount doctor acoustic avoid letter advice cage above"},
		{"ffffffffffffffffffffffffffffffff", "zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo wrong"},
	}
}

func TestNewRecoveryWords_MapsEntropyToBIP39Words(t *testing.T) {
	t.Parallel()
	for _, vector := range bip39Vectors() {
		entropy, err := hex.DecodeString(vector.entropy)
		if err != nil {
			t.Fatal(err)
		}
		words, err := identity.NewRecoveryWords(bytes.NewReader(entropy))
		if err != nil {
			t.Fatalf("NewRecoveryWords(%s) error = %v", vector.entropy, err)
		}
		if got := strings.Join(words.List(), " "); got != vector.words {
			t.Errorf("words of %s = %q, want %q", vector.entropy, got, vector.words)
		}
	}
}

func TestParseRecoveryWords_AcceptsTheBIP39Vectors(t *testing.T) {
	t.Parallel()
	for _, vector := range bip39Vectors() {
		parsed, err := identity.ParseRecoveryWords(strings.Fields(vector.words))
		if err != nil {
			t.Fatalf("ParseRecoveryWords(%q) error = %v", vector.words, err)
		}
		entropy, err := hex.DecodeString(vector.entropy)
		if err != nil {
			t.Fatal(err)
		}
		same, err := identity.NewRecoveryWords(bytes.NewReader(entropy))
		if err != nil {
			t.Fatal(err)
		}
		if deriveRecovery(t, parsed).Public().Fingerprint() != deriveRecovery(t, same).Public().Fingerprint() {
			t.Errorf("parsed words of %s give another key than the entropy", vector.entropy)
		}
	}
}

func TestNewRecoveryWords_ReturnsTwelveParsableWords(t *testing.T) {
	t.Parallel()
	words, err := identity.NewRecoveryWords(bytes.NewReader(bytes.Repeat([]byte{0x42}, 16)))
	if err != nil {
		t.Fatal(err)
	}
	if len(words.List()) != identity.WordCount || identity.WordCount != 12 {
		t.Fatalf("List() has %d words, WordCount = %d, want 12", len(words.List()), identity.WordCount)
	}
	if _, err := identity.ParseRecoveryWords(words.List()); err != nil {
		t.Errorf("a new set of words does not parse: %v", err)
	}
}

func TestNewRecoveryWords_FailsWhenTheRandomSourceFails(t *testing.T) {
	t.Parallel()
	if _, err := identity.NewRecoveryWords(iotest.ErrReader(errors.New("no entropy"))); err == nil {
		t.Error("NewRecoveryWords with a failing source returned no error")
	}
	if _, err := identity.NewRecoveryWords(bytes.NewReader(make([]byte, 15))); err == nil {
		t.Error("NewRecoveryWords with 15 bytes returned no error")
	}
}

func TestParseRecoveryWords_RefusesWordsWithAMistake(t *testing.T) {
	t.Parallel()
	abandon := strings.Fields(strings.Repeat("abandon ", 12))
	valid := strings.Fields(bip39Vectors()[0].words)
	withWord := func(position int, word string) []string {
		out := append([]string(nil), valid...)
		out[position-1] = word
		return out
	}
	tests := []struct {
		name  string
		words []string
		want  error
	}{
		{"twelve times abandon has a broken checksum", abandon, identity.ErrChecksum},
		{"eleven words", valid[:11], identity.ErrWordCount},
		{"thirteen words", append(append([]string(nil), valid...), "about"), identity.ErrWordCount},
		{"no words", nil, identity.ErrWordCount},
		{"a word that is not in the list", withWord(4, "numen"), identity.ErrUnknownWord},
		{"a four letter start of a word", withWord(4, "aban"), identity.ErrUnknownWord},
		{"an empty word", withWord(4, " "), identity.ErrUnknownWord},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := identity.ParseRecoveryWords(tt.words)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestParseRecoveryWords_NamesThePositionAndNotTheWord(t *testing.T) {
	t.Parallel()
	words := strings.Fields(bip39Vectors()[0].words)
	words[3] = "numen"
	_, err := identity.ParseRecoveryWords(words)
	if !errors.Is(err, identity.ErrUnknownWord) {
		t.Fatalf("error = %v, want ErrUnknownWord", err)
	}
	if !strings.Contains(err.Error(), "4") {
		t.Errorf("error %q does not name the position 4", err)
	}
	if strings.Contains(err.Error(), "numen") {
		t.Errorf("error %q echoes the word", err)
	}
}

func TestParseRecoveryWords_TrimsAndLowerCasesEachWord(t *testing.T) {
	t.Parallel()
	words := strings.Fields(bip39Vectors()[1].words)
	words[0] = "  LEGAL\n"
	words[5] = "Sausage"
	parsed, err := identity.ParseRecoveryWords(words)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if got := strings.Join(parsed.List(), " "); got != bip39Vectors()[1].words {
		t.Errorf("List() = %q, want %q", got, bip39Vectors()[1].words)
	}
}

func TestRecoveryWords_PrintNoWord(t *testing.T) {
	t.Parallel()
	words := parseWords(t, bip39Vectors()[1].words)
	if words.String() != "[recovery words]" {
		t.Errorf("String() = %q", words.String())
	}
	for _, text := range []string{
		fmt.Sprint(words), fmt.Sprintf("%+v", words), fmt.Sprintf("[%s]", words),
		fmt.Sprint(fmt.Errorf("failed with %v", words)),
	} {
		for _, word := range words.List() {
			if strings.Contains(text, word) {
				t.Errorf("%q prints the word %q", text, word)
			}
		}
	}
}

func TestParseRecoveryWords_ReturnsTheWordsOfAnyEntropy(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		entropy := rapid.SliceOfN(rapid.Byte(), 16, 16).Draw(t, "entropy")
		made, err := identity.NewRecoveryWords(bytes.NewReader(entropy))
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := identity.ParseRecoveryWords(made.List())
		if err != nil {
			t.Fatalf("the words %v do not parse: %v", made.List(), err)
		}
		if strings.Join(parsed.List(), " ") != strings.Join(made.List(), " ") {
			t.Fatalf("parsed words %v differ from %v", parsed.List(), made.List())
		}
		if deriveRecovery(t, parsed).Public().Fingerprint() != deriveRecovery(t, made).Public().Fingerprint() {
			t.Fatal("parsed words derive another key")
		}
	})
}
