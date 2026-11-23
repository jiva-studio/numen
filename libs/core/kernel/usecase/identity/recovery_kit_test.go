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
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/identity/slip39"
)

var kitIDFormat = regexp.MustCompile(`^[0-9A-F]{4}$`)

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

func TestStartRestoreFromKit_JoinsTheAccountOfTheSheetsOnAnotherDevice(t *testing.T) {
	t.Parallel()
	first := newRestoreKit(t, "Desktop")
	created, err := first.create.Execute(t.Context(), getLegalWords())
	if err != nil {
		t.Fatal(err)
	}
	made, err := first.createKit.Execute(t.Context(), getLegalWords(), nil)
	if err != nil {
		t.Fatal(err)
	}
	second := newRestoreKit(t, "Phone")

	if err := second.startKit.Execute(t.Context(), pick(t, made, 2, 3)); err != nil {
		t.Fatal(err)
	}

	pending, err := second.pending.Execute(t.Context())
	if err != nil || !pending.IsPending || !bytes.Equal(pending.RootSignKey, second.getRootSignKey(t, getLegalWords())) {
		t.Fatalf("pending = %+v, error = %v, want the root key of the words", pending, err)
	}
	if got := second.keys.snapshot(); len(got) != 0 {
		t.Fatalf("the key store holds %v while the restore is pending", got)
	}
	if err := second.complete.Execute(t.Context(), syncPlugin, first.getChain(t)); err != nil {
		t.Fatal(err)
	}
	got, err := second.get.Execute(t.Context())
	if err != nil || got.AccountFingerprint != created.AccountFingerprint || got.DeviceFingerprint == created.DeviceFingerprint || got.DeviceName != "Phone" {
		t.Errorf("identity = %+v, error = %v, want the account of the first device and a device of its own", got, err)
	}
}

func TestStartRestoreFromKit_CompletesOfflineAsTheAccountOfTheWords(t *testing.T) {
	t.Parallel()
	first := newRestoreKit(t, "Desktop")
	if _, err := first.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	made, err := first.createKit.Execute(t.Context(), getLegalWords(), nil)
	if err != nil {
		t.Fatal(err)
	}
	fresh := newRestoreKit(t, "Phone")
	if err := fresh.startKit.Execute(t.Context(), pick(t, made, 1, 3)); err != nil {
		t.Fatal(err)
	}

	fresh.clock.Advance(restoreFallback)

	want := accountOfWords(t, getLegalWords())
	testsupport.WaitFor(t, func() bool {
		got, err := fresh.get.Execute(t.Context())
		return err == nil && got.AccountFingerprint == want
	})
}

func TestStartRestoreFromKit_RefusesOneSheetAndNothingIsPending(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	fresh := newRestoreKit(t, "Phone")
	if err := fresh.startKit.Execute(t.Context(), pick(t, made, 1)); !errors.Is(err, identitydomain.ErrNotEnoughSheets) {
		t.Errorf("error = %v, want ErrNotEnoughSheets", err)
	}
	if pending, err := fresh.pending.Execute(t.Context()); err != nil || pending.IsPending {
		t.Errorf("pending = %+v, error = %v, want nothing pending", pending, err)
	}
}

func TestStartRestoreFromKit_RefusesASheetWithAMistakeAndNothingIsPending(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	fresh := newRestoreKit(t, "Phone")
	sheets := pick(t, made, 1, 3)
	sheets[1] = withWrongWord(sheets[1])
	err := fresh.startKit.Execute(t.Context(), sheets)
	if !errors.Is(err, identitydomain.ErrInvalidSheets) {
		t.Fatalf("error = %v, want ErrInvalidSheets", err)
	}
	var sheetErr *identitylib.SheetError
	if !errors.As(err, &sheetErr) || sheetErr.Position != 2 {
		t.Errorf("error %v does not name the position 2", err)
	}
	if pending, err := fresh.pending.Execute(t.Context()); err != nil || pending.IsPending {
		t.Errorf("pending = %+v, error = %v, want nothing pending", pending, err)
	}
}

func TestStartRestoreFromKit_RefusesWhenTheDeviceHasAnIdentity(t *testing.T) {
	t.Parallel()
	k := newRestoreKit(t, "Laptop")
	if _, err := k.create.Execute(t.Context(), getLegalWords()); err != nil {
		t.Fatal(err)
	}
	made, err := k.createKit.Execute(t.Context(), getLegalWords(), nil)
	if err != nil {
		t.Fatal(err)
	}
	before := k.keys.snapshot()[keyName]
	if err := k.startKit.Execute(t.Context(), pick(t, made, 1, 2)); !errors.Is(err, identitydomain.ErrIdentityExists) {
		t.Fatalf("error = %v, want ErrIdentityExists", err)
	}
	if k.keys.snapshot()[keyName] != before {
		t.Error("the record changed")
	}
	k.keys.keys[keyName] = "not-a-record"
	if err := k.startKit.Execute(t.Context(), pick(t, made, 1, 2)); !errors.Is(err, identitydomain.ErrIdentityExists) {
		t.Errorf("over a broken record: error = %v, want ErrIdentityExists", err)
	}
	if pending, err := k.pending.Execute(t.Context()); err != nil || pending.IsPending {
		t.Errorf("pending = %+v, error = %v, want nothing pending", pending, err)
	}
}

func TestRecoveryKit_KeepsNoWordNoSecretNoSheetInTheKeyStore(t *testing.T) {
	t.Parallel()
	k := newKit("Laptop")
	createdWith(t, k, getLegalWords())
	made := kitOf(t, k, getLegalWords())
	fresh := newRestoreKit(t, "Phone")
	if err := fresh.startKit.Execute(t.Context(), pick(t, made, 2, 3)); err != nil {
		t.Fatal(err)
	}
	if err := fresh.complete.Execute(t.Context(), syncPlugin, nil); err != nil {
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
	for name, value := range fresh.keys.snapshot() {
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
