package identity_test

import (
	"strings"
	"testing"

	"connectrpc.com/connect"

	windowidentity "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/identity"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func startRestore(t *testing.T, h *windowidentity.Handler, words []string) *v1.StartRestoreResponse {
	t.Helper()
	answer, err := h.StartRestore(t.Context(), connect.NewRequest(&v1.StartRestoreRequest{Words: words}))
	if err != nil {
		t.Fatalf("StartRestore() error = %v", err)
	}
	return answer.Msg
}

func TestStartRestore_AnswersWithoutAnErrorAndKeepsNoIdentityYet(t *testing.T) {
	t.Parallel()
	keys := newKeys()
	h := newHandler(keys)

	answer := startRestore(t, h, getLegalWords())

	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_UNSPECIFIED || answer.GetDetail() != "" {
		t.Errorf("answer = %v, want no error", answer)
	}
	if len(keys.keys) != 0 {
		t.Errorf("the key store holds %v, want nothing while the restore is pending", keys.keys)
	}
	got, err := h.GetIdentity(t.Context(), connect.NewRequest(&v1.GetIdentityRequest{}))
	if err != nil || got.Msg.GetIdentity() != nil {
		t.Errorf("GetIdentity() = %v, %v, want no identity until the restore completes", got, err)
	}
}

func TestStartRestore_RefusesWordsThatAreNotValidWithoutEchoingAWord(t *testing.T) {
	t.Parallel()
	h := newHandler(newKeys())
	words := getLegalWords()
	words[3] = "numen"

	answer := startRestore(t, h, words)

	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_WORDS {
		t.Fatalf("answer = %v, want the code of words that are not valid", answer)
	}
	for _, word := range append(words, "legal") {
		if strings.Contains(answer.GetDetail(), word) {
			t.Errorf("detail %q echoes the word %q", answer.GetDetail(), word)
		}
	}
	checksum := startRestore(t, h, strings.Fields(strings.Repeat("abandon ", 12)))
	if checksum.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_INVALID_WORDS || strings.Contains(checksum.GetDetail(), "abandon") {
		t.Errorf("checksum answer = %v", checksum)
	}
}

func TestStartRestore_RefusesADeviceThatHasAnIdentity(t *testing.T) {
	t.Parallel()
	h := newHandler(newKeys())
	createGenerated(t, h)

	answer := startRestore(t, h, getLegalWords())

	if answer.GetError() != v1.IdentityErrorCode_IDENTITY_ERROR_CODE_IDENTITY_EXISTS {
		t.Errorf("answer = %v, want the code of an identity that exists", answer)
	}
}

func TestNewService_AnswersUnimplementedForStartRestoreWhereTheInstallationHasNoKeyStore(t *testing.T) {
	t.Parallel()
	service := windowidentity.NewService(nil)

	_, err := service.StartRestore(t.Context(), connect.NewRequest(&v1.StartRestoreRequest{Words: getLegalWords()}))

	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("error = %v, want Unimplemented", err)
	}
}
