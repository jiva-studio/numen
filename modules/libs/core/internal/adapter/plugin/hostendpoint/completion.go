package hostendpoint

import (
	"errors"
	"strings"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/port/completionport"
)

func writeRefusal(err error) (v1.CompletionErrorCode, bool) {
	switch {
	case errors.Is(err, completionport.ErrNoConsent):
		return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_CONSENT, true
	case errors.Is(err, completionport.ErrNoRoute):
		return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE, true
	case errors.Is(err, completionport.ErrUnavailable):
		return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE, true
	}
	return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNSPECIFIED, false
}

func readWireQuestion(wire *v1.CompletionQuestion) completionport.Question {
	messages := make([]completionport.Message, len(wire.GetMessages()))
	for i, message := range wire.GetMessages() {
		messages[i] = completionport.Message{Role: message.GetRole(), Text: message.GetText()}
	}
	return completionport.Question{
		Messages: messages,
		Purpose:  wire.GetPurpose(),
	}
}

// already states.
func readRefusalDetail(err error) string {
	text := err.Error()
	for _, sentinel := range []error{completionport.ErrNoRoute, completionport.ErrUnavailable, completionport.ErrNoConsent} {
		if rest, found := strings.CutPrefix(text, sentinel.Error()); found {
			return strings.TrimPrefix(rest, ": ")
		}
	}
	return text
}
