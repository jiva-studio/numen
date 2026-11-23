package identity_test

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	windowidentity "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/identity"
	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

var kitID = regexp.MustCompile(`^[0-9A-F]{4}$`)

// kitSaver stands where the person saves the kit: it keeps the kits it is given, or answers that no
// file was chosen, or fails as a machine without a save dialog.
type kitSaver struct {
	mu        sync.Mutex
	kits      []identitydomain.Kit
	isRefused bool
	fails     error
}

func (s *kitSaver) SaveKit(_ context.Context, kit identitydomain.Kit) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fails != nil {
		return false, s.fails
	}
	if s.isRefused {
		return false, nil
	}
	s.kits = append(s.kits, kit)
	return true, nil
}

func (s *kitSaver) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.kits)
}

func (s *kitSaver) getLast(t *testing.T) identitydomain.Kit {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.kits) == 0 {
		t.Fatal("no kit was saved")
	}
	return s.kits[len(s.kits)-1]
}

func getOtherWords() []string {
	return strings.Fields("letter advice cage absurd amount doctor acoustic avoid letter advice cage above")
}

func wordsProof(words []string) *v1.SaveRecoveryKitRequest {
	return &v1.SaveRecoveryKitRequest{Proof: &v1.SaveRecoveryKitRequest_Words{Words: &v1.RecoveryWordsProof{Words: words}}}
}

func sheetsProof(sheets []*v1.RecoverySheet) *v1.SaveRecoveryKitRequest {
	return &v1.SaveRecoveryKitRequest{Proof: &v1.SaveRecoveryKitRequest_Sheets{Sheets: &v1.RecoverySheetsProof{Sheets: sheets}}}
}

func saveKit(t *testing.T, h *windowidentity.Handler, request *v1.SaveRecoveryKitRequest) *v1.SaveRecoveryKitResponse {
	t.Helper()
	answer, err := h.SaveRecoveryKit(t.Context(), connect.NewRequest(request))
	if err != nil {
		t.Fatalf("SaveRecoveryKit() error = %v", err)
	}
	return answer.Msg
}

func restoreKit(t *testing.T, h *windowidentity.Handler, sheets []*v1.RecoverySheet) *v1.RestoreIdentityFromKitResponse {
	t.Helper()
	answer, err := h.RestoreIdentityFromKit(t.Context(), connect.NewRequest(&v1.RestoreIdentityFromKitRequest{Sheets: sheets}))
	if err != nil {
		t.Fatalf("RestoreIdentityFromKit() error = %v", err)
	}
	return answer.Msg
}

func pickSheets(t *testing.T, made identitydomain.Kit, numbers ...int) []*v1.RecoverySheet {
	t.Helper()
	picked := make([]*v1.RecoverySheet, 0, len(numbers))
	for _, number := range numbers {
		if number < 1 || number > len(made.Sheets) {
			t.Fatalf("the kit has no sheet %d", number)
		}
		picked = append(picked, &v1.RecoverySheet{Words: slices.Clone(made.Sheets[number-1].Words)})
	}
	return picked
}

type kitFixture struct {
	handler *windowidentity.Handler
	keys    *keyStore
	saver   *kitSaver
	kit     identitydomain.Kit
}

func newKitFixture(t *testing.T) kitFixture {
	t.Helper()
	keys := newKeys()
	saver := &kitSaver{}
	h := newHandlerWith(keys, saver)
	create(t, h, getLegalWords())
	answer := saveKit(t, h, wordsProof(getLegalWords()))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED || !answer.GetIsSaved() {
		t.Fatalf("SaveRecoveryKit() = %v", answer)
	}
	return kitFixture{handler: h, keys: keys, saver: saver, kit: saver.getLast(t)}
}

func TestSaveRecoveryKit_SavesThreeSheetsOf33WordsForTheAccount(t *testing.T) {
	t.Parallel()
	f := newKitFixture(t)
	info, err := f.handler.GetIdentity(t.Context(), connect.NewRequest(&v1.GetIdentityRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !kitID.MatchString(f.kit.ID) {
		t.Errorf("kit id = %q, want 4 upper-case hex digits", f.kit.ID)
	}
	if f.kit.AccountFingerprint == "" || f.kit.AccountFingerprint != info.Msg.GetIdentity().GetAccountFingerprint() {
		t.Errorf("account fingerprint = %q, want the one of the identity", f.kit.AccountFingerprint)
	}
	if len(f.kit.Sheets) != 3 {
		t.Fatalf("kit has %d sheets, want 3", len(f.kit.Sheets))
	}
	for i, sheet := range f.kit.Sheets {
		if len(sheet.Words) != 33 {
			t.Errorf("sheet %d has %d words, want 33", i+1, len(sheet.Words))
		}
	}
}

func TestSaveRecoveryKit_AcceptsTwoSheetsAsTheProof(t *testing.T) {
	t.Parallel()
	f := newKitFixture(t)
	answer := saveKit(t, f.handler, sheetsProof(pickSheets(t, f.kit, 1, 3)))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED || !answer.GetIsSaved() || f.saver.count() != 2 {
		t.Errorf("answer = %v after %d kits", answer, f.saver.count())
	}
}

func TestSaveRecoveryKit_AnswersNotSavedWhenNoFileIsChosen(t *testing.T) {
	t.Parallel()
	f := newKitFixture(t)
	f.saver.isRefused = true
	answer := saveKit(t, f.handler, wordsProof(getLegalWords()))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED || answer.GetIsSaved() {
		t.Errorf("answer = %v", answer)
	}
}

func TestSaveRecoveryKit_AnswersUnavailableWithoutASaveDialog(t *testing.T) {
	t.Parallel()
	for _, failure := range []error{machine.ErrNoFolderDialog, machine.ErrChoosing} {
		keys := newKeys()
		h := newHandlerWith(keys, &kitSaver{fails: failure})
		create(t, h, getLegalWords())
		_, err := h.SaveRecoveryKit(t.Context(), connect.NewRequest(wordsProof(getLegalWords())))
		if connect.CodeOf(err) != connect.CodeUnavailable {
			t.Errorf("%v: error = %v, want Unavailable", failure, err)
		}
	}
}

func TestSaveRecoveryKit_WritesNothingToTheKeyStore(t *testing.T) {
	t.Parallel()
	f := newKitFixture(t)
	before := f.keys.keys["identity"]
	saveKit(t, f.handler, wordsProof(getLegalWords()))
	if len(f.keys.keys) != 1 || f.keys.keys["identity"] != before {
		t.Errorf("the key store changed: %d entries", len(f.keys.keys))
	}
}

func TestSaveRecoveryKit_RefusesWordsOfAnotherAccountAndSavesNothing(t *testing.T) {
	t.Parallel()
	f := newKitFixture(t)
	answer := saveKit(t, f.handler, wordsProof(getOtherWords()))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_OTHER_ACCOUNT || answer.GetIsSaved() || f.saver.count() != 1 {
		t.Fatalf("answer = %v after %d kits", answer, f.saver.count())
	}
	for _, word := range getOtherWords() {
		if strings.Contains(" "+answer.GetDetail()+" ", " "+word+" ") {
			t.Errorf("detail %q echoes the word %q", answer.GetDetail(), word)
		}
	}
}

func TestSaveRecoveryKit_RefusesWordsThatAreNotValid(t *testing.T) {
	t.Parallel()
	f := newKitFixture(t)
	words := getLegalWords()
	words[3] = "numen"
	answer := saveKit(t, f.handler, wordsProof(words))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_WORDS || answer.GetIsSaved() {
		t.Fatalf("answer = %v", answer)
	}
	if !strings.Contains(answer.GetDetail(), "4") || strings.Contains(answer.GetDetail(), "numen") {
		t.Errorf("detail = %q, want the position 4 and no word", answer.GetDetail())
	}
}

func TestSaveRecoveryKit_NamesTheSheetWithAMistakeAndEchoesNoWord(t *testing.T) {
	t.Parallel()
	f := newKitFixture(t)
	sheets := pickSheets(t, f.kit, 1, 2)
	changed := sheets[1].GetWords()
	changed[5] = "academic"
	if f.kit.Sheets[1].Words[5] == "academic" {
		changed[5] = "acid"
	}
	answer := saveKit(t, f.handler, sheetsProof(sheets))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_SHEETS {
		t.Fatalf("answer = %v", answer)
	}
	if answer.GetDetail() != "sheet 2 is not valid" {
		t.Errorf("detail = %q, want %q", answer.GetDetail(), "sheet 2 is not valid")
	}
}

func TestSaveRecoveryKit_RefusesWithoutAnIdentity(t *testing.T) {
	t.Parallel()
	answer := saveKit(t, newHandler(newKeys()), wordsProof(getLegalWords()))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NO_IDENTITY || answer.GetIsSaved() {
		t.Errorf("answer = %v", answer)
	}
}

func TestSaveRecoveryKit_AnswersInvalidArgumentWithoutAProofOrWithoutSheets(t *testing.T) {
	t.Parallel()
	h := newHandler(newKeys())
	for name, request := range map[string]*v1.SaveRecoveryKitRequest{
		"no proof":     {},
		"empty sheets": sheetsProof(nil),
	} {
		_, err := h.SaveRecoveryKit(t.Context(), connect.NewRequest(request))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s: error = %v, want InvalidArgument", name, err)
		}
	}
}

func TestRestoreIdentityFromKit_AnswersTheSameAccountAfterADelete(t *testing.T) {
	t.Parallel()
	f := newKitFixture(t)
	if _, err := f.handler.DeleteIdentity(t.Context(), connect.NewRequest(&v1.DeleteIdentityRequest{})); err != nil {
		t.Fatal(err)
	}
	answer := restoreKit(t, f.handler, pickSheets(t, f.kit, 2, 3))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("answer = %v", answer)
	}
	info := answer.GetIdentity()
	if info.GetAccountFingerprint() != f.kit.AccountFingerprint || info.GetDeviceName() != "Laptop" || info.GetDeviceFingerprint() == "" {
		t.Errorf("identity = %v", info)
	}
	if len(f.keys.keys) != 1 {
		t.Errorf("the key store holds %d entries, want 1", len(f.keys.keys))
	}
}

func TestRestoreIdentityFromKit_RefusesOneSheetAndKeepsNothing(t *testing.T) {
	t.Parallel()
	f := newKitFixture(t)
	if _, err := f.handler.DeleteIdentity(t.Context(), connect.NewRequest(&v1.DeleteIdentityRequest{})); err != nil {
		t.Fatal(err)
	}
	answer := restoreKit(t, f.handler, pickSheets(t, f.kit, 1))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NOT_ENOUGH_SHEETS || answer.GetIdentity() != nil || len(f.keys.keys) != 0 {
		t.Errorf("answer = %v, keys = %v", answer, f.keys.keys)
	}
}

func TestRestoreIdentityFromKit_NamesTheSheetWithAMistake(t *testing.T) {
	t.Parallel()
	f := newKitFixture(t)
	if _, err := f.handler.DeleteIdentity(t.Context(), connect.NewRequest(&v1.DeleteIdentityRequest{})); err != nil {
		t.Fatal(err)
	}
	sheets := pickSheets(t, f.kit, 1, 3)
	sheets[1].GetWords()[0] = "academic"
	if f.kit.Sheets[2].Words[0] == "academic" {
		sheets[1].GetWords()[0] = "acid"
	}
	answer := restoreKit(t, f.handler, sheets)
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_SHEETS || answer.GetDetail() != "sheet 2 is not valid" || len(f.keys.keys) != 0 {
		t.Errorf("answer = %v, keys = %v", answer, f.keys.keys)
	}
}

func TestRestoreIdentityFromKit_RefusesOverAnExistingIdentityAndAKeyStoreFailure(t *testing.T) {
	t.Parallel()
	f := newKitFixture(t)
	if answer := restoreKit(t, f.handler, pickSheets(t, f.kit, 1, 2)); answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_IDENTITY_EXISTS {
		t.Errorf("over an identity: answer = %v", answer)
	}
	if _, err := f.handler.DeleteIdentity(t.Context(), connect.NewRequest(&v1.DeleteIdentityRequest{})); err != nil {
		t.Fatal(err)
	}
	f.keys.setFails = errors.New("the keychain refuses")
	if answer := restoreKit(t, f.handler, pickSheets(t, f.kit, 1, 2)); answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_KEY_STORE_FAILED || len(f.keys.keys) != 0 {
		t.Errorf("failing key store: answer = %v, keys = %v", answer, f.keys.keys)
	}
}

func TestRestoreIdentityFromKit_AnswersInvalidArgumentWithoutSheets(t *testing.T) {
	t.Parallel()
	_, err := newHandler(newKeys()).RestoreIdentityFromKit(t.Context(), connect.NewRequest(&v1.RestoreIdentityFromKitRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("error = %v, want InvalidArgument", err)
	}
}

func TestNewService_AnswersUnimplementedForTheKitCallsWithoutAKeyStore(t *testing.T) {
	t.Parallel()
	service := windowidentity.NewService(nil)
	if _, err := service.SaveRecoveryKit(t.Context(), connect.NewRequest(wordsProof(getLegalWords()))); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("SaveRecoveryKit error = %v, want Unimplemented", err)
	}
	if _, err := service.RestoreIdentityFromKit(t.Context(), connect.NewRequest(&v1.RestoreIdentityFromKitRequest{})); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("RestoreIdentityFromKit error = %v, want Unimplemented", err)
	}
}
