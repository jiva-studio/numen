package service

import (
	"context"
	"errors"
	"unicode"
	"unicode/utf8"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/sourcehealth"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/usecase/serving"
)

// maxHealthText is the longest text the host takes for an item.
const maxHealthText = 120

// observeSource takes the outcome of a call of the model: a precondition the settings do not meet (no
// connection, a connection that fails, a local model that is missing) reports the item, a call that
// succeeded clears it, and any other failure leaves it as it is.
func observeSource(ctx context.Context, reporter *sourcehealth.Reporter, err error) {
	failure, isPrecondition := errors.AsType[serving.PreconditionError](err)
	switch {
	case err == nil:
		reporter.Set(ctx, "")
	case isPrecondition:
		reporter.Set(ctx, getHealthText(failure.Error()))
	}
}

// getHealthText is the message as a sentence that fits an item: its first letter upper case, cut at the
// limit of the host.
func getHealthText(message string) string {
	if message == "" {
		return ""
	}
	first, size := utf8.DecodeRuneInString(message)
	text := string(unicode.ToUpper(first)) + message[size:]
	if runes := []rune(text); len(runes) > maxHealthText {
		text = string(runes[:maxHealthText-1]) + "…"
	}
	return text
}
