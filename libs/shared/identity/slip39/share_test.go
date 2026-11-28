package slip39_test

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity/slip39"
)

func validMnemonic(t *testing.T) []string {
	t.Helper()
	return strings.Fields(readVectors(t)[22].Mnemonics[0])
}

func differentWord(tb testing.TB, list []string, word string) string {
	tb.Helper()
	for _, candidate := range list {
		if candidate != word {
			return candidate
		}
	}
	tb.Fatal("the word list has one word")
	return ""
}

func TestParseShare_ReadsAMnemonicOf33Words(t *testing.T) {
	t.Parallel()
	words := validMnemonic(t)
	if len(words) != 33 {
		t.Fatalf("the vector has %d words, want 33", len(words))
	}
	share, err := slip39.ParseShare(words)
	if err != nil {
		t.Fatal(err)
	}
	if len(share.Value) != 32 {
		t.Errorf("value has %d bytes, want 32", len(share.Value))
	}
	if got := share.Words(); strings.Join(got, " ") != strings.Join(words, " ") {
		t.Errorf("Words() = %v, want %v", got, words)
	}
}

func TestParseShare_TrimsAndLowerCasesEachWord(t *testing.T) {
	t.Parallel()
	words := validMnemonic(t)
	noisy := make([]string, len(words))
	for i, word := range words {
		noisy[i] = "  " + strings.ToUpper(word) + "\t"
	}
	got, err := slip39.ParseShare(noisy)
	if err != nil {
		t.Fatal(err)
	}
	want, err := slip39.ParseShare(words)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Value, want.Value) || got.Identifier != want.Identifier {
		t.Error("the noisy words parse to another share")
	}
}

func TestParseShare_RefusesAWrongWordCount(t *testing.T) {
	t.Parallel()
	words := validMnemonic(t)
	for _, count := range []int{0, 1, 19, 32} {
		if _, err := slip39.ParseShare(words[:count]); !errors.Is(err, slip39.ErrWordCount) {
			t.Errorf("%d words: error = %v, want ErrWordCount", count, err)
		}
	}
	if _, err := slip39.ParseShare(append(slices.Clone(words), "academic")); !errors.Is(err, slip39.ErrWordCount) {
		t.Error("34 words are accepted")
	}
}

func TestParseShare_RefusesAnUnknownWordAndNamesItsPosition(t *testing.T) {
	t.Parallel()
	words := validMnemonic(t)
	words[6] = "notaword"
	_, err := slip39.ParseShare(words)
	if !errors.Is(err, slip39.ErrUnknownWord) {
		t.Fatalf("error = %v, want ErrUnknownWord", err)
	}
	if !strings.Contains(err.Error(), "7") {
		t.Errorf("error %q does not name the position 7", err)
	}
	if strings.Contains(err.Error(), "notaword") {
		t.Errorf("error %q echoes the word", err)
	}
}

func TestParseShare_RefusesAWordOfTheBIP39ListThatIsNotInTheSLIP39List(t *testing.T) {
	t.Parallel()
	words := validMnemonic(t)
	words[0] = "abandon"
	if _, err := slip39.ParseShare(words); !errors.Is(err, slip39.ErrUnknownWord) {
		t.Errorf("error = %v, want ErrUnknownWord", err)
	}
}

func TestParseShare_RefusesOneChangedWordByChecksum(t *testing.T) {
	t.Parallel()
	list := readWordList(t)
	for at := range 33 {
		words := validMnemonic(t)
		words[at] = differentWord(t, list, words[at])
		if _, err := slip39.ParseShare(words); !errors.Is(err, slip39.ErrChecksum) {
			t.Errorf("word %d changed: error = %v, want ErrChecksum", at+1, err)
		}
	}
}

func TestParseShare_RefusesTwoSwappedWords(t *testing.T) {
	t.Parallel()
	words := validMnemonic(t)
	words[3], words[9] = words[9], words[3]
	if _, err := slip39.ParseShare(words); !errors.Is(err, slip39.ErrChecksum) {
		t.Errorf("error = %v, want ErrChecksum", err)
	}
}

func TestShare_PrintsAPlaceholderInsteadOfTheValue(t *testing.T) {
	t.Parallel()
	share, err := slip39.ParseShare(validMnemonic(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%v", share); got != "[slip39 share]" {
		t.Errorf("%%v = %q, want [slip39 share]", got)
	}
	for _, text := range []string{fmt.Sprintf("%+v", share), fmt.Sprintf("%#v", share), fmt.Sprint(share)} {
		for _, word := range share.Words() {
			if strings.Contains(text, " "+word+" ") {
				t.Errorf("%q prints the word %q", text, word)
			}
		}
		if strings.Contains(text, fmt.Sprint(share.Value)) {
			t.Errorf("%q prints the value", text)
		}
	}
}
