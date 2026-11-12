package service

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/engine"
)

func TestOpenerReturnsMockRecognizer(t *testing.T) {
	isMock := func() bool { return true }
	opened, err := newOpener(engine.NewLibrary(), isMock)(t.Context(), settings.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if _, isMockRecognizer := opened.(engine.MockRecognizer); !isMockRecognizer {
		t.Fatalf("opened %T, want the mock", opened)
	}
}
