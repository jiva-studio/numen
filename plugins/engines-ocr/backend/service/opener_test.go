package service

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/settings"
)

func TestOpenerReturnsMockRecogniser(t *testing.T) {
	isMock := func() bool { return true }
	opened, err := newOpener(engine.NewLibrary(), isMock)(t.Context(), settings.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if _, isMockRecogniser := opened.(engine.MockRecogniser); !isMockRecogniser {
		t.Fatalf("opened %T, want the mock", opened)
	}
}
