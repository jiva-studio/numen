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

func observeSource(ctx context.Context, reporter *sourcehealth.Reporter, err error) {
	failure, isPrecondition := errors.AsType[serving.PreconditionError](err)
	switch {
	case err == nil:
		reporter.Set(ctx, "")
	case isPrecondition:
		reporter.Set(ctx, getHealthText(failure.Error()))
	}
}

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
