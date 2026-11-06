package engine

import (
	"context"
	"errors"
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
	recognise []string
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
	recognise, err := locateRecognisers(cfg)
	if err != nil {
		return nil, "", paths{}, err
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

// locateRecognisers finds the custom model where one is configured, else every file the settings name.
func locateRecognisers(cfg Config) ([]string, error) {
	if cfg.Recognise.Path != "" {
		found, err := locateModel(cfg.Recognise.Path, "", "", cfg.Dir)
		if err != nil {
			return nil, fmt.Errorf("recognise model: %w", err)
		}
		return []string{found}, nil
	}
	if len(cfg.Recognise.Files) == 0 {
		return nil, errors.New("recognise model: the settings name none")
	}
	found := make([]string, 0, len(cfg.Recognise.Files))
	for _, file := range cfg.Recognise.Files {
		pos, err := locateModel("", file.EnvVar, file.FileName, cfg.Dir)
		if err != nil {
			return nil, fmt.Errorf("recognise model %s: %w", file.ID, err)
		}
		found = append(found, pos)
	}
	return found, nil
}
