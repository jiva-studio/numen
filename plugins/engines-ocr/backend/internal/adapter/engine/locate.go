package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"

	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
)

// EnvDir names the folder plugins keep their models in. The host sets it, to the folder a user chose or
// to models in the shared data directory.
const EnvDir = "NUMEN_MODELS_DIR"

type paths struct {
	layout    string
	detect    string
	recognize string
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

// FetchModelFile resolves a pinned model through the host.
type FetchModelFile func(context.Context, string, string) (string, error)

type modelFiles struct {
	dir            string
	shouldDownload bool
	fetch          FetchModelFile
}

func (files modelFiles) resolve(ctx context.Context, path, address, envVar, name string) (string, error) {
	if address == settings.RecognizerCustom && path == "" {
		return "", fmt.Errorf("a custom recognizer needs a model path")
	}
	local, err := locateModel(path, envVar, name, files.dir)
	if err == nil || path != "" || !files.shouldDownload {
		return local, err
	}
	digest, ok := settings.FindModelDigest(address)
	if !ok {
		return "", fmt.Errorf("no SHA-256 pinned for model %q", address)
	}
	return files.fetch(ctx, address, digest)
}

func locateModels(ctx context.Context, library *onnx.Library, cfg settings.Config, fetch FetchModelFile) (*ort.Engine, string, paths, error) {
	engine, pos, err := library.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, cfg.Dir))
	if err != nil {
		return nil, "", paths{}, err
	}
	files := modelFiles{dir: cfg.Dir, shouldDownload: cfg.ShouldDownload, fetch: fetch}
	layout, err := files.resolve(ctx, cfg.Layout.Path, cfg.Layout.Name, "NUMEN_DEPENDENCY_PP_DOCLAYOUT", "pp-doclayout.onnx")
	if err != nil {
		return nil, "", paths{}, fmt.Errorf("layout model: %w", err)
	}
	detect, err := files.resolve(ctx, cfg.Detect.Path, cfg.Detect.Name, "NUMEN_DEPENDENCY_PP_OCR_DET", "pp-ocr-det.onnx")
	if err != nil {
		return nil, "", paths{}, fmt.Errorf("detect model: %w", err)
	}
	recognize, err := files.resolve(ctx, cfg.Recognize.Path, cfg.Recognize.Name, "NUMEN_DEPENDENCY_PP_OCR_REC", "pp-ocr-rec.onnx")
	if err != nil {
		return nil, "", paths{}, fmt.Errorf("recognize model: %w", err)
	}

	found := paths{
		layout:    layout,
		detect:    detect,
		recognize: recognize,
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

func getCacheRoot() string {
	root, err := cache.GetSharedDirOrPluginDir()
	if err != nil {
		return ""
	}
	return root
}

func isPresent(pos string) bool {
	_, err := os.Stat(pos)
	return err == nil
}
