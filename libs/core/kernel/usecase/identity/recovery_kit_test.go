package identity_test

import (
	"bytes"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/identity/slip39"
)

var kitIDFormat = regexp.MustCompile(`^[0-9A-F]{4}$`)

func getOtherWords() []string {
	return strings.Fields("letter advice cage absurd amount doctor acoustic avoid letter advice cage above")
}

func createdWith(t *testing.T, k kit, words []string) identitydomain.Identity {
	t.Helper()
	created, err := k.create.Execute(t.Context(), words)
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	return created
}

func kitOf(t *testing.T, k kit, words []string) identitydomain.Kit {
	t.Helper()
	made, err := k.createKit.Execute(t.Context(), words, nil)
	if err != nil {
		t.Fatalf("create recovery kit: %v", err)
	}
	return made
}

func pick(t *testing.T, made identitydomain.Kit, numbers ...int) [][]string {
	t.Helper()
	picked := make([][]string, 0, len(numbers))
	for _, number := range numbers {
		if number < 1 || number > len(made.Sheets) {
			t.Fatalf("the kit has no sheet %d", number)
		}
		picked = append(picked, slices.Clone(made.Sheets[number-1].Words))
	}
	return picked
}

func withWrongWord(sheet []string) []string {
	changed := slices.Clone(sheet)
	if changed[5] == "academic" {
		changed[5] = "acid"
	} else {
		changed[5] = "academic"
	}
	return changed
}

func accountOfWords(t *testing.T, words []string) string {
	t.Helper()
	parsed, err := identitylib.ParseRecoveryWords(words)
	if err != nil {
		t.Fatal(err)
	}
	key, err := parsed.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	return key.Public().Fingerprint().String()
}

func TestCreateRecoveryKit_MakesThreeSheetsForTheAccountOfTheWords(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	created := createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	if !kitIDFormat.MatchString(made.ID) {
		t.Errorf("kit id = %q, want 4 upper-case hex digits", made.ID)
	}
	if made.AccountFingerprint != created.AccountFingerprint {
		t.Errorf("kit account = %q, want %q", made.AccountFingerprint, created.AccountFingerprint)
	}
	if len(made.Sheets) != 3 {
		t.Fatalf("kit has %d sheets, want 3", len(made.Sheets))
	}
	for i, sheet := range made.Sheets {
		if sheet.Number != i+1 || len(sheet.Words) != 33 {
			t.Errorf("sheet %d: number %d with %d words, want %d with 33", i, sheet.Number, len(sheet.Words), i+1)
		}
		if _, err := identitylib.ParseSheet(sheet.Words); err != nil {
			t.Errorf("sheet %d does not parse: %v", i+1, err)
		}
	}
}

func TestCreateRecoveryKit_MakesSheetsThatRestoreTheAccountFromAnyTwo(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	created := createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	for _, pair := range [][]int{{1, 2}, {1, 3}, {2, 3}, {3, 1}} {
		got, err := k.checkKit.Execute(t.Context(), pick(t, made, pair...))
		if err != nil {
			t.Fatalf("check sheets %v: %v", pair, err)
		}
		if got != created.AccountFingerprint {
			t.Errorf("sheets %v fit %q, want %q", pair, got, created.AccountFingerprint)
		}
	}
}

func TestCreateRecoveryKit_WritesNothingToTheKeyStore(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	before := k.keys.snapshot()
	writes := k.keys.setCalled
	made := kitOf(t, k, getLegalWords())
	if _, err := k.createKit.Execute(t.Context(), nil, pick(t, made, 1, 2)); err != nil {
		t.Fatal(err)
	}
	after := k.keys.snapshot()
	if len(after) != len(before) || after[keyName] != before[keyName] || k.keys.setCalled != writes {
		t.Errorf("the key store changed: %d writes after %d", k.keys.setCalled, writes)
	}
}

func TestCreateRecoveryKit_ProvesTheAccountWithTwoSheetsOfAnEarlierKit(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	created := createdWith(t, k, getLegalWords())
	first := kitOf(t, k, getLegalWords())
	second, err := k.createKit.Execute(t.Context(), nil, pick(t, first, 2, 3))
	if err != nil {
		t.Fatalf("create from sheets: %v", err)
	}
	if second.AccountFingerprint != created.AccountFingerprint || len(second.Sheets) != 3 {
		t.Errorf("second kit = account %q with %d sheets", second.AccountFingerprint, len(second.Sheets))
	}
	if second.ID == first.ID && slices.Equal(second.Sheets[0].Words, first.Sheets[0].Words) {
		t.Error("a new kit has the sheets of the old kit")
	}
}

func TestCreateRecoveryKit_ANewKitDoesNotCancelAnOlderKit(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	created := createdWith(t, k, getLegalWords())
	older := kitOf(t, k, getLegalWords())
	kitOf(t, k, getLegalWords())
	got, err := k.checkKit.Execute(t.Context(), pick(t, older, 1, 3))
	if err != nil || got != created.AccountFingerprint {
		t.Errorf("the older kit: %q, %v; want %q", got, err, created.AccountFingerprint)
	}
}

func TestCreateRecoveryKit_RefusesWordsOfAnotherAccountAndShowsNoSheet(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made, err := k.createKit.Execute(t.Context(), getOtherWords(), nil)
	if !errors.Is(err, identitydomain.ErrOtherAccount) {
		t.Fatalf("error = %v, want ErrOtherAccount", err)
	}
	if len(made.Sheets) != 0 || made.ID != "" || made.AccountFingerprint != "" {
		t.Errorf("a refusal returned a kit: %+v", made)
	}
}

func TestCreateRecoveryKit_RefusesSheetsOfAnotherAccount(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	stranger := newKit("Phone")
	createdWith(t, stranger, getOtherWords())
	foreign := kitOf(t, stranger, getOtherWords())
	made, err := k.createKit.Execute(t.Context(), nil, pick(t, foreign, 1, 2))
	if !errors.Is(err, identitydomain.ErrOtherAccount) {
		t.Fatalf("error = %v, want ErrOtherAccount", err)
	}
	if len(made.Sheets) != 0 {
		t.Error("a refusal returned sheets")
	}
}

func TestCreateRecoveryKit_RefusesAnInvalidProofAndShowsNoSheet(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	unknown := slices.Clone(getLegalWords())
	unknown[3] = "numen"

	_, err := k.createKit.Execute(t.Context(), unknown, nil)
	if !errors.Is(err, identitydomain.ErrInvalidWords) {
		t.Errorf("unknown word: error = %v, want ErrInvalidWords", err)
	}
	if err != nil && strings.Contains(err.Error(), "numen") {
		t.Errorf("the error echoes a word: %v", err)
	}
	_, err = k.createKit.Execute(t.Context(), nil, [][]string{withWrongWord(pick(t, made, 1)[0]), pick(t, made, 2)[0]})
	if !errors.Is(err, identitydomain.ErrInvalidSheets) {
		t.Errorf("wrong sheet: error = %v, want ErrInvalidSheets", err)
	}
	_, err = k.createKit.Execute(t.Context(), nil, pick(t, made, 1))
	if !errors.Is(err, identitydomain.ErrNotEnoughSheets) {
		t.Errorf("one sheet: error = %v, want ErrNotEnoughSheets", err)
	}
}

func TestCreateRecoveryKit_RefusesNoProofOrTwoProofs(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	if _, err := k.createKit.Execute(t.Context(), nil, nil); err == nil {
		t.Error("no proof makes a kit")
	}
	if _, err := k.createKit.Execute(t.Context(), getLegalWords(), pick(t, made, 1, 2)); err == nil {
		t.Error("words and sheets together make a kit")
	}
}

func TestCreateRecoveryKit_RefusesWithoutAnIdentity(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	if _, err := k.createKit.Execute(t.Context(), getLegalWords(), nil); !errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("error = %v, want ErrNoIdentity", err)
	}
}

func TestCreateRecoveryKit_ReportsAKeyStoreFailure(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	k.keys.getFails = errBroken
	_, err := k.createKit.Execute(t.Context(), getLegalWords(), nil)
	if !errors.Is(err, identitydomain.ErrKeyStore) || !errors.Is(err, errBroken) {
		t.Errorf("error = %v, want ErrKeyStore wrapping the failure", err)
	}
}

func TestCheckRecoveryKit_RefusesOneSheet(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	for _, sheets := range [][][]string{nil, pick(t, made, 2)} {
		got, err := k.checkKit.Execute(t.Context(), sheets)
		if !errors.Is(err, identitydomain.ErrNotEnoughSheets) {
			t.Errorf("%d sheets: error = %v, want ErrNotEnoughSheets", len(sheets), err)
		}
		if errors.Is(err, identitydomain.ErrInvalidSheets) {
			t.Errorf("%d sheets: a missing sheet is reported as an invalid sheet", len(sheets))
		}
		if got != "" {
			t.Errorf("%d sheets: fingerprint %q returned with a refusal", len(sheets), got)
		}
	}
}

func TestCheckRecoveryKit_NamesTheSheetWithAMistakeAndEchoesNoWord(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	sheets := pick(t, made, 1, 2)
	sheets[1] = withWrongWord(sheets[1])
	_, err := k.checkKit.Execute(t.Context(), sheets)
	if !errors.Is(err, identitydomain.ErrInvalidSheets) {
		t.Fatalf("error = %v, want ErrInvalidSheets", err)
	}
	var sheetErr *identitylib.SheetError
	if !errors.As(err, &sheetErr) || sheetErr.Position != 2 {
		t.Errorf("error %v does not name the position 2", err)
	}
	for _, word := range sheets[1] {
		if strings.Contains(err.Error(), " "+word+" ") {
			t.Errorf("the error echoes the word %q: %v", word, err)
		}
	}
}

func TestCheckRecoveryKit_RefusesTheSameSheetTwiceAndSheetsOfTwoKits(t *testing.T) {
	t.Parallel()
	words := getLegalWords()
	k := newKit("Laptop")
	createdWith(t, k, words)
	first := kitOf(t, k, words)
	second := kitOf(t, k, words)
	twice := [][]string{pick(t, first, 2)[0], pick(t, first, 2)[0]}
	if _, err := k.checkKit.Execute(t.Context(), twice); !errors.Is(err, identitydomain.ErrInvalidSheets) {
		t.Errorf("the same sheet twice: error = %v, want ErrInvalidSheets", err)
	}
	mixed := [][]string{pick(t, first, 1)[0], pick(t, second, 2)[0]}
	if _, err := k.checkKit.Execute(t.Context(), mixed); !errors.Is(err, identitydomain.ErrInvalidSheets) {
		t.Errorf("sheets of two kits: error = %v, want ErrInvalidSheets", err)
	}
}

func TestCheckRecoveryKit_RefusesSheetsOfAnotherAccount(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	stranger := newKit("Phone")
	createdWith(t, stranger, getOtherWords())
	foreign := kitOf(t, stranger, getOtherWords())
	got, err := k.checkKit.Execute(t.Context(), pick(t, foreign, 1, 3))
	if !errors.Is(err, identitydomain.ErrOtherAccount) {
		t.Errorf("error = %v, want ErrOtherAccount", err)
	}
	if got != "" {
		t.Errorf("fingerprint %q returned with a refusal", got)
	}
}

func TestCheckRecoveryKit_RefusesWithoutAnIdentityAndReportsAKeyStoreFailure(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	sheets := pick(t, made, 1, 2)
	k.keys.getFails = errBroken
	if _, err := k.checkKit.Execute(t.Context(), sheets); !errors.Is(err, identitydomain.ErrKeyStore) {
		t.Errorf("a refused read: error = %v, want ErrKeyStore", err)
	}
	k.keys.getFails = nil
	if err := k.delete.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := k.checkKit.Execute(t.Context(), sheets); !errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("no record: error = %v, want ErrNoIdentity", err)
	}
}

func TestRestoreIdentityFromKit_RestoresTheSameAccountOnAnotherDeviceKey(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	created := createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	if err := k.delete.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	writes := k.keys.setCalled
	restored, err := k.restoreKit.Execute(t.Context(), pick(t, made, 2, 3))
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.AccountFingerprint != created.AccountFingerprint {
		t.Errorf("account = %q, want %q", restored.AccountFingerprint, created.AccountFingerprint)
	}
	if restored.DeviceFingerprint == created.DeviceFingerprint {
		t.Error("the restored identity has the old device key")
	}
	if restored.DeviceName != "Laptop" || !restored.DeviceAddedAt.Equal(getNow()) {
		t.Errorf("device = %q added %v", restored.DeviceName, restored.DeviceAddedAt)
	}
	if got := k.keys.snapshot(); len(got) != 1 || k.keys.setCalled != writes+1 {
		t.Errorf("key store = %d entries after %d writes, want one entry written once", len(got), k.keys.setCalled-writes)
	}
	got, err := k.get.Execute(t.Context())
	if err != nil || got != restored {
		t.Errorf("GetIdentity() = %+v, %v; want %+v", got, err, restored)
	}
}

func TestRestoreIdentityFromKit_OnAFreshDeviceMatchesTheAccountOfTheWords(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	fresh := newKit("Phone")
	restored, err := fresh.restoreKit.Execute(t.Context(), pick(t, made, 1, 3))
	if err != nil {
		t.Fatal(err)
	}
	if want := accountOfWords(t, getLegalWords()); restored.AccountFingerprint != want {
		t.Errorf("account = %q, want %q of the words", restored.AccountFingerprint, want)
	}
	if err := fresh.delete.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	again := createdWith(t, fresh, getLegalWords())
	if again.AccountFingerprint != restored.AccountFingerprint {
		t.Error("the words give another account than the sheets")
	}
}

func TestRestoreIdentityFromKit_RefusesOneSheetAndWritesNothing(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	if err := k.delete.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	writes := k.keys.setCalled
	if _, err := k.restoreKit.Execute(t.Context(), pick(t, made, 1)); !errors.Is(err, identitydomain.ErrNotEnoughSheets) {
		t.Errorf("error = %v, want ErrNotEnoughSheets", err)
	}
	if got := k.keys.snapshot(); len(got) != 0 || k.keys.setCalled != writes {
		t.Errorf("the key store holds %v after %d more writes", got, k.keys.setCalled-writes)
	}
}

func TestRestoreIdentityFromKit_RefusesASheetWithAMistakeAndWritesNothing(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	if err := k.delete.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	sheets := pick(t, made, 1, 3)
	sheets[1] = withWrongWord(sheets[1])
	_, err := k.restoreKit.Execute(t.Context(), sheets)
	if !errors.Is(err, identitydomain.ErrInvalidSheets) {
		t.Fatalf("error = %v, want ErrInvalidSheets", err)
	}
	var sheetErr *identitylib.SheetError
	if !errors.As(err, &sheetErr) || sheetErr.Position != 2 {
		t.Errorf("error %v does not name the position 2", err)
	}
	if got := k.keys.snapshot(); len(got) != 0 {
		t.Errorf("the key store holds %v", got)
	}
}

func TestRestoreIdentityFromKit_NeverOverwritesARecord(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	before := k.keys.snapshot()[keyName]
	if _, err := k.restoreKit.Execute(t.Context(), pick(t, made, 1, 2)); !errors.Is(err, identitydomain.ErrIdentityExists) {
		t.Fatalf("error = %v, want ErrIdentityExists", err)
	}
	if k.keys.snapshot()[keyName] != before || k.keys.setCalled != 1 {
		t.Error("the record changed")
	}
	k.keys.keys[keyName] = "not-a-record"
	if _, err := k.restoreKit.Execute(t.Context(), pick(t, made, 1, 2)); !errors.Is(err, identitydomain.ErrIdentityExists) {
		t.Errorf("over a broken record: error = %v, want ErrIdentityExists", err)
	}
	if k.keys.keys[keyName] != "not-a-record" {
		t.Error("the broken record was overwritten")
	}
}

func TestRestoreIdentityFromKit_ReportsAKeyStoreThatRefusesAndKeepsNothing(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	if err := k.delete.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	k.keys.setFails = errBroken
	_, err := k.restoreKit.Execute(t.Context(), pick(t, made, 1, 2))
	if !errors.Is(err, identitydomain.ErrKeyStore) || !errors.Is(err, errBroken) {
		t.Fatalf("error = %v, want ErrKeyStore wrapping the failure", err)
	}
	if got := k.keys.snapshot(); len(got) != 0 {
		t.Errorf("the key store holds %v", got)
	}
}

func TestRecoveryKit_KeepsNoWordNoSecretNoSheetInTheKeyStore(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	if err := k.delete.Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := k.restoreKit.Execute(t.Context(), pick(t, made, 2, 3)); err != nil {
		t.Fatal(err)
	}
	entropy := bytes.Repeat([]byte{0x7f}, 16)
	secret, err := hkdf.Key(sha256.New, entropy, nil, "numen/v1/recovery", 32)
	if err != nil {
		t.Fatal(err)
	}
	var shareBytes [][]byte
	for _, sheet := range made.Sheets {
		share, err := slip39.ParseShare(sheet.Words)
		if err != nil {
			t.Fatal(err)
		}
		shareBytes = append(shareBytes, share.Value)
	}
	for name, value := range k.keys.snapshot() {
		raw, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			t.Fatal(err)
		}
		for _, sheet := range made.Sheets {
			for _, word := range slices.Concat(sheet.Words, getLegalWords()) {
				if bytes.Contains(raw, []byte(word)) {
					t.Errorf("%s holds the word %q", name, word)
				}
			}
		}
		for _, secretLike := range append([][]byte{secret}, shareBytes...) {
			if bytes.Contains(raw, secretLike) {
				t.Errorf("%s holds a recovery secret or a share value", name)
			}
		}
	}
}
