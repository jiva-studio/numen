package hostendpoint

import (
	"errors"
	"strings"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/completion"
)

func writeRefusal(err error) (v1.CompletionErrorCode, bool) {
	switch {
	case errors.Is(err, completion.ErrNoRoute):
		return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE, true
	case errors.Is(err, domain.ErrUnavailable):
		return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE, true
	}
	return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNSPECIFIED, false
}

func readWireQuestion(wire *v1.CompletionQuestion) completion.Question {
	messages := make([]completion.Message, len(wire.GetMessages()))
	for i, message := range wire.GetMessages() {
		messages[i] = completion.Message{Role: message.GetRole(), Text: message.GetText()}
	}
	return completion.Question{
		Messages: messages,
		Purpose:  wire.GetPurpose(),
	}
}

func readRefusalDetail(err error) string {
	text := err.Error()
	for _, sentinel := range []error{completion.ErrNoRoute, domain.ErrUnavailable} {
		if rest, found := strings.CutPrefix(text, sentinel.Error()); found {
			return strings.TrimPrefix(rest, ": ")
		}
	}
	return text
}
