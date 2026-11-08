package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/onnx"
)

// EnvDir names the folder plugins keep their models in. The host sets it, to the folder a user chose or
// to models in the shared data directory.
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
	if base, err := cache.GetSharedDataDirOrPluginDataDir(); err == nil {
		return findInFolder(filepath.Join(base, "models"), fileName)
	}
	return ""
}

func locateModel(explicitPath, envVar, fileName, dir string) (string, error) {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err != nil {
			return "", fmt.Errorf("configured model path not found: %w", err)
		}
		return explicitPath, nil
	}
	if envVar != "" {
		if env := os.Getenv(envVar); isPresent(env) {
			return env, nil
		}
	}
	if found := findInFolder(dir, fileName); found != "" {
		return found, nil
	}
	if found := findInHome(fileName); found != "" {
		return found, nil
	}
	return "", fmt.Errorf("model %s not found", fileName)
}

func locateModels(ctx context.Context, library *onnx.Library, cfg Config) (*ort.Engine, string, paths, error) {
	engine, pos, err := library.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, cfg.Dir))
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
