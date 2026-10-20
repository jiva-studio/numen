package plugin

import (
	"errors"
	"strings"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// writeRefusal maps an error of a completion to the code a refused call answers
// with. It reports false for an error that is no refusal.
func writeRefusal(err error) (v1.CompletionErrorCode, bool) {
	switch {
	case errors.Is(err, port.ErrNoConsent):
		return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_CONSENT, true
	case errors.Is(err, port.ErrNoRoute):
		return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE, true
	case errors.Is(err, port.ErrUnavailable):
		return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE, true
	}
	return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNSPECIFIED, false
}

func readWireQuestion(wire *v1.CompletionQuestion) port.Question {
	messages := make([]port.Message, len(wire.GetMessages()))
	for i, message := range wire.GetMessages() {
		messages[i] = port.Message{Role: message.GetRole(), Text: message.GetText()}
	}
	return port.Question{
		Messages: messages,
		Purpose:  wire.GetPurpose(),
	}
}

// readRefusalDetail is the text of a refusal without the sentinel that names its code, which the code
// already states.
func readRefusalDetail(err error) string {
	text := err.Error()
	for _, sentinel := range []error{port.ErrNoRoute, port.ErrUnavailable, port.ErrNoConsent} {
		if rest, found := strings.CutPrefix(text, sentinel.Error()); found {
			return strings.TrimPrefix(rest, ": ")
		}
	}
	return text
}
