package identity_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	windowidentity "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

var kitID = regexp.MustCompile(`^[0-9A-F]{4}$`)

func getOtherWords() []string {
	return strings.Fields("letter advice cage absurd amount doctor acoustic avoid letter advice cage above")
}

func wordsProof(words []string) *v1.CreateRecoveryKitRequest {
	return &v1.CreateRecoveryKitRequest{Proof: &v1.CreateRecoveryKitRequest_Words{Words: &v1.RecoveryWordsProof{Words: words}}}
}

func sheetsProof(sheets []*v1.RecoverySheet) *v1.CreateRecoveryKitRequest {
	return &v1.CreateRecoveryKitRequest{Proof: &v1.CreateRecoveryKitRequest_Sheets{Sheets: &v1.RecoverySheetsProof{Sheets: sheets}}}
}

func createKit(t *testing.T, h *windowidentity.Handler, request *v1.CreateRecoveryKitRequest) *v1.CreateRecoveryKitResponse {
	t.Helper()
	answer, err := h.CreateRecoveryKit(t.Context(), connect.NewRequest(request))
	if err != nil {
		t.Fatalf("CreateRecoveryKit() error = %v", err)
	}
	return answer.Msg
}

func checkKit(t *testing.T, h *windowidentity.Handler, sheets []*v1.RecoverySheet) *v1.CheckRecoveryKitResponse {
	t.Helper()
	answer, err := h.CheckRecoveryKit(t.Context(), connect.NewRequest(&v1.CheckRecoveryKitRequest{Sheets: sheets}))
	if err != nil {
		t.Fatalf("CheckRecoveryKit() error = %v", err)
	}
	return answer.Msg
}

func restoreKit(t *testing.T, h *windowidentity.Handler, sheets []*v1.RecoverySheet) *v1.StartRestoreFromKitResponse {
	t.Helper()
	answer, err := h.StartRestoreFromKit(t.Context(), connect.NewRequest(&v1.StartRestoreFromKitRequest{Sheets: sheets}))
	if err != nil {
		t.Fatalf("StartRestoreFromKit() error = %v", err)
	}
	return answer.Msg
}

func pickSheets(t *testing.T, made *v1.RecoveryKit, numbers ...int) []*v1.RecoverySheet {
	t.Helper()
	picked := make([]*v1.RecoverySheet, 0, len(numbers))
	for _, number := range numbers {
		sheets := made.GetSheets()
		if number < 1 || number > len(sheets) {
			t.Fatalf("the kit has no sheet %d", number)
		}
		picked = append(picked, &v1.RecoverySheet{Words: slices.Clone(sheets[number-1].GetWords())})
	}
	return picked
}

func newKitFixture(t *testing.T) (*windowidentity.Handler, *keyStore, *v1.RecoveryKit) {
	t.Helper()
	keys := newKeys()
	h := newHandler(keys)
	create(t, h, getLegalWords())
	answer := createKit(t, h, wordsProof(getLegalWords()))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED || answer.GetKit() == nil {
		t.Fatalf("CreateRecoveryKit() = %v", answer)
	}
	return h, keys, answer.GetKit()
}

func TestCreateRecoveryKit_AnswersThreeSheetsOf33WordsForTheWords(t *testing.T) {
	t.Parallel()
	h, _, made := newKitFixture(t)
	info, err := h.GetIdentity(t.Context(), connect.NewRequest(&v1.GetIdentityRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !kitID.MatchString(made.GetKitId()) {
		t.Errorf("kit_id = %q, want 4 upper-case hex digits", made.GetKitId())
	}
	if made.GetAccountFingerprint() == "" || made.GetAccountFingerprint() != info.Msg.GetIdentity().GetAccountFingerprint() {
		t.Errorf("account_fingerprint = %q, want the one of the identity", made.GetAccountFingerprint())
	}
	if len(made.GetSheets()) != 3 {
		t.Fatalf("kit has %d sheets, want 3", len(made.GetSheets()))
	}
	for i, sheet := range made.GetSheets() {
		if len(sheet.GetWords()) != 33 {
			t.Errorf("sheet %d has %d words, want 33", i+1, len(sheet.GetWords()))
		}
	}
}

func TestCreateRecoveryKit_AcceptsTwoSheetsAsTheProof(t *testing.T) {
	t.Parallel()
	h, _, made := newKitFixture(t)
	answer := createKit(t, h, sheetsProof(pickSheets(t, made, 1, 3)))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED || len(answer.GetKit().GetSheets()) != 3 {
		t.Errorf("answer = %v", answer)
	}
}

func TestCreateRecoveryKit_WritesNothingToTheKeyStore(t *testing.T) {
	t.Parallel()
	h, keys, _ := newKitFixture(t)
	before := keys.keys["identity"]
	createKit(t, h, wordsProof(getLegalWords()))
	if len(keys.keys) != 1 || keys.keys["identity"] != before {
		t.Errorf("the key store changed: %d entries", len(keys.keys))
	}
}

func TestCreateRecoveryKit_RefusesWordsOfAnotherAccountWithoutASheet(t *testing.T) {
	t.Parallel()
	h, _, _ := newKitFixture(t)
	answer := createKit(t, h, wordsProof(getOtherWords()))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_OTHER_ACCOUNT || answer.GetKit() != nil {
		t.Fatalf("answer = %v", answer)
	}
	for _, word := range getOtherWords() {
		if strings.Contains(" "+answer.GetDetail()+" ", " "+word+" ") {
			t.Errorf("detail %q echoes the word %q", answer.GetDetail(), word)
		}
	}
}

func TestCreateRecoveryKit_RefusesWordsThatAreNotValid(t *testing.T) {
	t.Parallel()
	h, _, _ := newKitFixture(t)
	words := getLegalWords()
	words[3] = "numen"
	answer := createKit(t, h, wordsProof(words))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_WORDS || answer.GetKit() != nil {
		t.Fatalf("answer = %v", answer)
	}
	if !strings.Contains(answer.GetDetail(), "4") || strings.Contains(answer.GetDetail(), "numen") {
		t.Errorf("detail = %q, want the position 4 and no word", answer.GetDetail())
	}
}

func TestCreateRecoveryKit_RefusesWithoutAnIdentity(t *testing.T) {
	t.Parallel()
	answer := createKit(t, newHandler(newKeys()), wordsProof(getLegalWords()))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NO_IDENTITY || answer.GetKit() != nil {
		t.Errorf("answer = %v", answer)
	}
}

func TestCreateRecoveryKit_AnswersInvalidArgumentWithoutAProofOrWithoutSheets(t *testing.T) {
	t.Parallel()
	h := newHandler(newKeys())
	for name, request := range map[string]*v1.CreateRecoveryKitRequest{
		"no proof":     {},
		"empty sheets": sheetsProof(nil),
	} {
		_, err := h.CreateRecoveryKit(t.Context(), connect.NewRequest(request))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s: error = %v, want InvalidArgument", name, err)
		}
	}
}

func TestCheckRecoveryKit_AnswersTheFingerprintForAnyTwoSheets(t *testing.T) {
	t.Parallel()
	h, _, made := newKitFixture(t)
	for _, pair := range [][]int{{1, 2}, {1, 3}, {2, 3}} {
		answer := checkKit(t, h, pickSheets(t, made, pair...))
		if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED || answer.GetAccountFingerprint() != made.GetAccountFingerprint() {
			t.Errorf("sheets %v: answer = %v", pair, answer)
		}
	}
}

func TestCheckRecoveryKit_AnswersOneMoreSheetIsNeeded(t *testing.T) {
	t.Parallel()
	h, _, made := newKitFixture(t)
	answer := checkKit(t, h, pickSheets(t, made, 2))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NOT_ENOUGH_SHEETS || answer.GetAccountFingerprint() != "" {
		t.Errorf("answer = %v", answer)
	}
}

func TestCheckRecoveryKit_NamesTheSheetWithAMistakeAndEchoesNoWord(t *testing.T) {
	t.Parallel()
	h, _, made := newKitFixture(t)
	sheets := pickSheets(t, made, 1, 2)
	changed := sheets[1].GetWords()
	changed[5] = "academic"
	if made.GetSheets()[1].GetWords()[5] == "academic" {
		changed[5] = "acid"
	}
	answer := checkKit(t, h, sheets)
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_SHEETS {
		t.Fatalf("answer = %v", answer)
	}
	if answer.GetDetail() != "sheet 2 is not valid" {
		t.Errorf("detail = %q, want %q", answer.GetDetail(), "sheet 2 is not valid")
	}
}

func TestCheckRecoveryKit_RefusesSheetsOfAnotherAccount(t *testing.T) {
	t.Parallel()
	h, _, _ := newKitFixture(t)
	stranger := newHandler(newKeys())
	create(t, stranger, getOtherWords())
	foreign := createKit(t, stranger, wordsProof(getOtherWords())).GetKit()
	answer := checkKit(t, h, pickSheets(t, foreign, 1, 2))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_OTHER_ACCOUNT || answer.GetAccountFingerprint() != "" {
		t.Errorf("answer = %v", answer)
	}
}

func TestCheckRecoveryKit_AnswersNoIdentityAndInvalidArgument(t *testing.T) {
	t.Parallel()
	h, _, made := newKitFixture(t)
	empty := newHandler(newKeys())
	if answer := checkKit(t, empty, pickSheets(t, made, 1, 2)); answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NO_IDENTITY {
		t.Errorf("no identity: answer = %v", answer)
	}
	_, err := h.CheckRecoveryKit(t.Context(), connect.NewRequest(&v1.CheckRecoveryKitRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("empty sheets: error = %v, want InvalidArgument", err)
	}
}

func TestStartRestoreFromKit_RestoresTheSameAccountAfterADelete(t *testing.T) {
	t.Parallel()
	keys := newKeys()
	clock := testsupport.NewClockAt(time.Date(2026, time.November, 22, 12, 0, 0, 0, time.UTC))
	h := newHandlerAt(keys, clock)
	create(t, h, getLegalWords())
	made := createKit(t, h, wordsProof(getLegalWords())).GetKit()
	if _, err := h.DeleteIdentity(t.Context(), connect.NewRequest(&v1.DeleteIdentityRequest{})); err != nil {
		t.Fatal(err)
	}
	answer := restoreKit(t, h, pickSheets(t, made, 2, 3))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED || len(keys.keys) != 0 {
		t.Fatalf("answer = %v, keys = %v, want no error and nothing kept while the restore is pending", answer, keys.keys)
	}

	clock.Advance(30 * time.Second)

	testsupport.WaitFor(t, func() bool {
		got, err := h.GetIdentity(t.Context(), connect.NewRequest(&v1.GetIdentityRequest{}))
		info := got.Msg.GetIdentity()
		return err == nil && info.GetAccountFingerprint() == made.GetAccountFingerprint() && info.GetDeviceName() == "Laptop"
	})
}

func TestStartRestoreFromKit_RefusesOneSheetAndKeepsNothing(t *testing.T) {
	t.Parallel()
	h, keys, made := newKitFixture(t)
	if _, err := h.DeleteIdentity(t.Context(), connect.NewRequest(&v1.DeleteIdentityRequest{})); err != nil {
		t.Fatal(err)
	}
	answer := restoreKit(t, h, pickSheets(t, made, 1))
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_NOT_ENOUGH_SHEETS || len(keys.keys) != 0 {
		t.Errorf("answer = %v, keys = %v", answer, keys.keys)
	}
}

func TestStartRestoreFromKit_NamesTheSheetWithAMistake(t *testing.T) {
	t.Parallel()
	h, keys, made := newKitFixture(t)
	if _, err := h.DeleteIdentity(t.Context(), connect.NewRequest(&v1.DeleteIdentityRequest{})); err != nil {
		t.Fatal(err)
	}
	sheets := pickSheets(t, made, 1, 3)
	sheets[1].GetWords()[0] = "academic"
	if made.GetSheets()[2].GetWords()[0] == "academic" {
		sheets[1].GetWords()[0] = "acid"
	}
	answer := restoreKit(t, h, sheets)
	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_SHEETS || answer.GetDetail() != "sheet 2 is not valid" || len(keys.keys) != 0 {
		t.Errorf("answer = %v, keys = %v", answer, keys.keys)
	}
}

func TestStartRestoreFromKit_RefusesOverAnExistingIdentity(t *testing.T) {
	t.Parallel()
	h, _, made := newKitFixture(t)
	if answer := restoreKit(t, h, pickSheets(t, made, 1, 2)); answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_IDENTITY_EXISTS {
		t.Errorf("over an identity: answer = %v", answer)
	}
}

func TestStartRestoreFromKit_AnswersInvalidArgumentWithoutSheets(t *testing.T) {
	t.Parallel()
	_, err := newHandler(newKeys()).StartRestoreFromKit(t.Context(), connect.NewRequest(&v1.StartRestoreFromKitRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("error = %v, want InvalidArgument", err)
	}
}

func TestNewService_AnswersUnimplementedForTheKitCallsWithoutAKeyStore(t *testing.T) {
	t.Parallel()
	service := windowidentity.NewService(nil)
	if _, err := service.CreateRecoveryKit(t.Context(), connect.NewRequest(wordsProof(getLegalWords()))); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("CreateRecoveryKit error = %v, want Unimplemented", err)
	}
	if _, err := service.CheckRecoveryKit(t.Context(), connect.NewRequest(&v1.CheckRecoveryKitRequest{})); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("CheckRecoveryKit error = %v, want Unimplemented", err)
	}
	if _, err := service.StartRestoreFromKit(t.Context(), connect.NewRequest(&v1.StartRestoreFromKitRequest{})); connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("StartRestoreFromKit error = %v, want Unimplemented", err)
	}
}
