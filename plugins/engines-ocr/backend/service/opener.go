package service

import (
	"context"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/engines"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/onnx"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/settings"
)

func newOpener(library *onnx.Library, isMock func() bool) engines.Opener {
	return func(ctx context.Context, cfg settings.Config) (engines.Recogniser, error) {
		if isMock() {
			return engine.MockRecogniser{}, nil
		}
		return engine.Open(ctx, library, cfg)
	}
}
