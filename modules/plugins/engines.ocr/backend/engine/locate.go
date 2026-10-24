package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
)

// EnvDir names the folder plugins keep their models in.
const EnvDir = "NUMEN_MODELS_DIR"

type paths struct {
	layout    string
	detect    string
	recognise string
	from      string
}

// locateModel finds a model file by checking:
// 1. Explicit path in settings
// 2. Provisioned environment variable (NUMEN_DEPENDENCY_*)
// 3. Plugin directory / models folder fallback
func locateModel(explicitPath, envVar, fileName, dir string) (string, error) {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err != nil {
			return "", fmt.Errorf("configured model path not found: %w", err)
		}
		return explicitPath, nil
	}
	if envVar != "" {
		if env := os.Getenv(envVar); env != "" {
			if _, err := os.Stat(env); err == nil {
				return env, nil
			}
		}
	}
	if dir != "" {
		candidate := filepath.Join(dir, fileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	if home := os.Getenv("NUMEN_MODELS_DIR"); home != "" {
		candidate := filepath.Join(home, fileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	if base, err := os.UserConfigDir(); err == nil {
		candidate := filepath.Join(base, "numen", "models", fileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("model %s not found (env: %s)", fileName, envVar)
}

func locateModels(ctx context.Context, cfg Config) (*ort.Engine, string, paths, error) {
	engine, pos, err := onnx.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, cfg.Dir))
	if err != nil {
		return nil, "", paths{}, err
	}
	layout, err := locateModel(cfg.Layout.Path, "NUMEN_DEPENDENCY_PP_DOCLAYOUT", "pp-doclayout.onnx", cfg.Dir)
	if err != nil {
		return nil, "", paths{}, fmt.Errorf("layout model: %w", err)
	}
	detect, err := locateModel(cfg.Detect.Path, "NUMEN_DEPENDENCY_PP_OCR_DET", "pp-ocr-det.onnx", cfg.Dir)
	if err != nil {
		return nil, "", paths{}, fmt.Errorf("detect model: %w", err)
	}
	recognise, err := locateModel(cfg.Recognise.Path, "NUMEN_DEPENDENCY_PP_OCR_REC", "pp-ocr-rec.onnx", cfg.Dir)
	if err != nil {
		return nil, "", paths{}, fmt.Errorf("recognise model: %w", err)
	}

	found := paths{
		layout:    layout,
		detect:    detect,
		recognise: recognise,
		from:      "provisioned",
	}
	if cfg.Dir != "" {
		found.from = cfg.Dir
	}
	return engine, pos, found, nil
}

func name(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}
