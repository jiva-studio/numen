package service

import (
	"context"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/engine"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/onnx"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/engines"
)

func newOpener(library *onnx.Library, isMock func() bool) engines.Opener {
	return func(ctx context.Context, cfg settings.Config) (engines.Recogniser, error) {
		if isMock() {
			return engine.MockRecogniser{}, nil
		}
		return engine.Open(ctx, library, cfg)
	}
}
