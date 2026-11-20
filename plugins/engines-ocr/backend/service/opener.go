package service

import (
	"context"

	modelprocess "github.com/jiva-studio/numen/libs/plugin/sdk-go/process"

	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/engine"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/engines"
)

func newOpener(library *onnx.Library, isMock func() bool, fetch engine.FetchModelFile, sessions *modelprocess.ModelSessions) engines.Opener {
	return func(ctx context.Context, cfg settings.Config) (engines.Recognizer, error) {
		if isMock() {
			return engine.MockRecognizer{}, nil
		}
		compute := sessions.GetCompute()
		cfg.Compute.Provider, cfg.Compute.ThreadLimit = compute.Provider, compute.ThreadLimit
		model, err := engine.Open(ctx, library, cfg, fetch)
		if err != nil {
			return nil, err
		}
		return activeRecognizer{Recognizer: model, release: sessions.Track(ctx, map[string]string{"Layout": string(model.LayoutProvider), "OCR detection": "cpu", "OCR recognition": "cpu"})}, nil
	}
}

type activeRecognizer struct {
	engines.Recognizer
	release func()
}

func (m activeRecognizer) Close() error {
	defer m.release()
	return m.Recognizer.Close()
}
