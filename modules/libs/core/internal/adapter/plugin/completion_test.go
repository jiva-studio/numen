package plugin

import (
	"errors"
	"fmt"
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The question a plugin sends is read with its messages and its purpose.
func TestAQuestionIsReadWithItsMessagesAndPurpose(t *testing.T) {
	got := readWireQuestion(&v1.CompletionQuestion{
		Messages: []*v1.CompletionMessage{{Role: "system", Text: "be brief"}, {Role: "user", Text: "hi"}},
		Purpose:  "proofreading",
	})
	if got.Purpose != "proofreading" || len(got.Messages) != 2 || got.Messages[0] != (port.Message{Role: "system", Text: "be brief"}) {
		t.Errorf("read %+v", got)
	}
	if bare := readWireQuestion(nil); len(bare.Messages) != 0 {
		t.Errorf("read %+v", bare)
	}
}

// A refusal is answered with its code, consent before route before unavailability, and any other
// error is none.
func TestARefusalIsAnsweredWithItsCode(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want v1.CompletionErrorCode
	}{
		{port.ErrNoConsent, v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_CONSENT},
		{port.ErrNoRoute, v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE},
		{fmt.Errorf("%w: down", port.ErrUnavailable), v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE},
	} {
		if got, isRefusal := writeRefusal(tc.err); !isRefusal || got != tc.want {
			t.Errorf("%v: got %v, %v; want %v", tc.err, got, isRefusal, tc.want)
		}
	}
	if _, isRefusal := writeRefusal(errors.New("other")); isRefusal {
		t.Error("an error of no refusal was answered with a code")
	}
}
