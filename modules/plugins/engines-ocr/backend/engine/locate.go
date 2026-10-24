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

func findInFolder(dir, fileName string) string {
	if dir == "" || fileName == "" {
		return ""
	}
	candidate := filepath.Join(dir, fileName)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func findInHome(fileName string) string {
	if home := os.Getenv(EnvDir); home != "" {
		if found := findInFolder(home, fileName); found != "" {
			return found
		}
	}
	if base, err := os.UserConfigDir(); err == nil {
		return findInFolder(filepath.Join(base, "numen", "models"), fileName)
	}
	return ""
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
		if env := os.Getenv(envVar); env != "" && isPresent(env) {
			return env, nil
		}
	}
	if found := findInFolder(dir, fileName); found != "" {
		return found, nil
	}
	if found := findInHome(fileName); found != "" {
		return found, nil
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
