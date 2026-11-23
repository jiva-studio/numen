package identity_test

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/iotest"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/identity/slip39"
)

var kitIDFormat = regexp.MustCompile(`^[0-9A-F]{4}$`)

func streamOf(seed byte) *bytes.Reader {
	return bytes.NewReader(bytes.Repeat([]byte{seed}, 4096))
}

func newKit(tb testing.TB, seed byte, words identity.RecoveryWords) identity.Kit {
	tb.Helper()
	kit, err := identity.NewKit(streamOf(seed), words.DeriveRecoverySecret())
	if err != nil {
		tb.Fatalf("NewKit error = %v", err)
	}
	return kit
}

func restoreKey(tb testing.TB, sheets ...identity.Sheet) identity.RecoveryPublicKey {
	tb.Helper()
	secret, err := identity.RestoreSecret(sheets)
	if err != nil {
		tb.Fatalf("RestoreSecret error = %v", err)
	}
	key, err := secret.DeriveRecoveryKey()
	if err != nil {
		tb.Fatal(err)
	}
	return key.Public()
}

func TestNewKit_MakesThreeSheetsOf33WordsInOrder(t *testing.T) {
	t.Parallel()
	kit := newKit(t, 0x21, abandonWords(t))
	if len(kit.Sheets) != identity.SheetCount || identity.SheetCount != 3 {
		t.Fatalf("a kit has %d sheets, want 3", len(kit.Sheets))
	}
	for i, sheet := range kit.Sheets {
		if got := sheet.Number(); got != i+1 {
			t.Errorf("sheet %d has number %d", i, got)
		}
		words := sheet.Words()
		if len(words) != identity.SheetWordCount || identity.SheetWordCount != 33 {
			t.Errorf("sheet %d has %d words, want 33", i+1, len(words))
		}
		parsed, err := identity.ParseSheet(words)
		if err != nil {
			t.Fatalf("sheet %d does not parse: %v", i+1, err)
		}
		if parsed.Number() != sheet.Number() || !slices.Equal(parsed.Words(), words) {
			t.Errorf("sheet %d changes when it is parsed", i+1)
		}
	}
}

func TestNewKit_NamesTheKitByFourUpperCaseHexDigits(t *testing.T) {
	t.Parallel()
	kit := newKit(t, 0x21, abandonWords(t))
	if !kitIDFormat.MatchString(kit.ID) {
		t.Errorf("kit id = %q, want 4 upper-case hex digits", kit.ID)
	}
	other := newKit(t, 0x93, abandonWords(t))
	if other.ID == kit.ID && slices.Equal(other.Sheets[0].Words(), kit.Sheets[0].Words()) {
		t.Error("two kits from two random streams are the same")
	}
}

func TestNewKit_FailsWhenTheRandomSourceFails(t *testing.T) {
	t.Parallel()
	secret := abandonWords(t).DeriveRecoverySecret()
	if _, err := identity.NewKit(iotest.ErrReader(errors.New("no entropy")), secret); err == nil {
		t.Error("NewKit ignores a failing random source")
	}
}

func TestRestoreSecret_RestoresTheKeyOfTheWordsFromAnyTwoSheets(t *testing.T) {
	t.Parallel()
	words := abandonWords(t)
	want := deriveRecovery(t, words).Public()
	kit := newKit(t, 0x21, words)
	for _, pair := range [][2]int{{0, 1}, {0, 2}, {1, 2}, {2, 0}} {
		got := restoreKey(t, kit.Sheets[pair[0]], kit.Sheets[pair[1]])
		if !bytes.Equal(got.Sign, want.Sign) || got.Box != want.Box {
			t.Errorf("sheets %v restore another recovery key", pair)
		}
		if got.Fingerprint() != want.Fingerprint() {
			t.Errorf("sheets %v restore another account fingerprint", pair)
		}
	}
}

func TestRestoreSecret_UsesTheFirstTwoOfThreeSheets(t *testing.T) {
	t.Parallel()
	words := abandonWords(t)
	kit := newKit(t, 0x21, words)
	got := restoreKey(t, kit.Sheets[0], kit.Sheets[1], kit.Sheets[2])
	if got.Fingerprint() != deriveRecovery(t, words).Public().Fingerprint() {
		t.Error("three sheets restore another account")
	}
}

func TestRestoreSecret_RefusesFewerThanTwoSheets(t *testing.T) {
	t.Parallel()
	kit := newKit(t, 0x21, abandonWords(t))
	for _, sheets := range [][]identity.Sheet{nil, {}, kit.Sheets[:1]} {
		if _, err := identity.RestoreSecret(sheets); !errors.Is(err, identity.ErrNotEnoughSheets) {
			t.Errorf("%d sheets: error = %v, want ErrNotEnoughSheets", len(sheets), err)
		}
	}
}

func TestRestoreSecret_RefusesTheSameSheetTwice(t *testing.T) {
	t.Parallel()
	kit := newKit(t, 0x21, abandonWords(t))
	_, err := identity.RestoreSecret([]identity.Sheet{kit.Sheets[1], kit.Sheets[1]})
	if !errors.Is(err, identity.ErrSheetInvalid) {
		t.Errorf("error = %v, want ErrSheetInvalid", err)
	}
}

func TestRestoreSecret_RefusesSheetsOfTwoKits(t *testing.T) {
	t.Parallel()
	words := abandonWords(t)
	first := newKit(t, 0x21, words)
	second := newKit(t, 0x93, words)
	_, err := identity.RestoreSecret([]identity.Sheet{first.Sheets[0], second.Sheets[1]})
	if !errors.Is(err, identity.ErrSheetInvalid) {
		t.Fatalf("error = %v, want ErrSheetInvalid", err)
	}
	var sheetErr *identity.SheetError
	if !errors.As(err, &sheetErr) {
		t.Fatalf("error %v does not carry a *SheetError", err)
	}
	if sheetErr.Position < 1 || sheetErr.Position > 2 {
		t.Errorf("position = %d, want 1 or 2", sheetErr.Position)
	}
}

func TestParseSheet_RefusesAChangedWordWithTheSLIP39Cause(t *testing.T) {
	t.Parallel()
	words := slices.Clone(newKit(t, 0x21, abandonWords(t)).Sheets[1].Words())
	words[4] = changedWord(words[4])
	_, err := identity.ParseSheet(words)
	if !errors.Is(err, identity.ErrSheetInvalid) {
		t.Fatalf("error = %v, want ErrSheetInvalid", err)
	}
	if !errors.Is(err, slip39.ErrChecksum) {
		t.Errorf("error %v does not keep the cause ErrChecksum", err)
	}
	if strings.Contains(err.Error(), words[4]) {
		t.Errorf("error %q echoes a word", err)
	}
}

func changedWord(word string) string {
	if word == "academic" {
		return "acid"
	}
	return "academic"
}

func TestParseSheet_RefusesWordsOfTheRecoveryList(t *testing.T) {
	t.Parallel()
	_, err := identity.ParseSheet(strings.Fields("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"))
	if !errors.Is(err, identity.ErrSheetInvalid) {
		t.Errorf("error = %v, want ErrSheetInvalid", err)
	}
}

func TestSheetAndSecret_PrintPlaceholdersWithoutWords(t *testing.T) {
	t.Parallel()
	words := abandonWords(t)
	kit := newKit(t, 0x21, words)
	secret := words.DeriveRecoverySecret()
	if got := secret.String(); got != "[recovery secret]" {
		t.Errorf("secret String = %q", got)
	}
	if got := kit.Sheets[0].String(); got != "[recovery sheet]" {
		t.Errorf("sheet String = %q", got)
	}
	for _, text := range []string{
		fmt.Sprintf("%v", kit.Sheets[0]), fmt.Sprintf("%+v", kit.Sheets[0]), fmt.Sprintf("%#v", kit.Sheets[0]),
		fmt.Sprintf("%v", secret), fmt.Sprintf("%+v", secret), fmt.Sprintf("%#v", secret),
	} {
		for _, word := range kit.Sheets[0].Words() {
			if strings.Contains(text, " "+word+" ") {
				t.Errorf("%q prints the word %q", text, word)
			}
		}
	}
}

func TestDeriveRecoverySecret_ReachesTheKeyOfTheWords(t *testing.T) {
	t.Parallel()
	words := abandonWords(t)
	viaSecret, err := words.DeriveRecoverySecret().DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	direct := deriveRecovery(t, words)
	if !bytes.Equal(viaSecret.Public().Sign, direct.Public().Sign) || viaSecret.Public().Box != direct.Public().Box {
		t.Error("the secret of the words derives another recovery key than the words")
	}
}

func TestRestoreSecret_RestoresTheSecretOfAnyWords(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		entropy := rapid.SliceOfN(rapid.Byte(), 16, 16).Draw(t, "entropy")
		stream := rapid.SliceOfN(rapid.Byte(), 4096, 4096).Draw(t, "stream")
		words, err := identity.NewRecoveryWords(bytes.NewReader(entropy))
		if err != nil {
			t.Fatal(err)
		}
		kit, err := identity.NewKit(bytes.NewReader(stream), words.DeriveRecoverySecret())
		if err != nil {
			t.Fatal(err)
		}
		picked := rapid.Permutation(kit.Sheets[:]).Draw(t, "order")[:2]
		secret, err := identity.RestoreSecret(picked)
		if err != nil {
			t.Fatal(err)
		}
		got, err := secret.DeriveRecoveryKey()
		if err != nil {
			t.Fatal(err)
		}
		if got.Public().Fingerprint() != deriveRecovery(t, words).Public().Fingerprint() {
			t.Fatal("two sheets restore another account")
		}
	})
}
