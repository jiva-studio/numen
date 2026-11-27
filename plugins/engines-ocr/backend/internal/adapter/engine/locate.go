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

func locateModel(explicitPath, fileName, dir string) (string, error) {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err != nil {
			return "", fmt.Errorf("configured model path not found: %w", err)
		}
		return explicitPath, nil
	}
	if found := findInFolder(dir, fileName); found != "" {
		return found, nil
	}
	return "", fmt.Errorf("model %s not found", fileName)
}

// FetchDependency resolves a model that the manifest declares under an id through the host.
type FetchDependency func(ctx context.Context, id string) (string, error)

type modelFiles struct {
	dir            string
	shouldDownload bool
	fetch          FetchDependency
}

func (files modelFiles) resolve(ctx context.Context, path, id, name string) (string, error) {
	if id == settings.RecognizerCustom && path == "" {
		return "", fmt.Errorf("a custom recognizer needs a model path")
	}
	local, err := locateModel(path, name, files.dir)
	if err == nil || path != "" || !files.shouldDownload {
		return local, err
	}
	return files.fetch(ctx, id)
}

func locateModels(ctx context.Context, library *onnx.Library, cfg settings.Config, fetch FetchDependency) (*ort.Engine, string, paths, error) {
	opened, err := library.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, cfg.Dir))
	if err != nil {
		return nil, "", paths{}, err
	}
	engine, pos := opened.Engine, opened.LibPath
	files := modelFiles{dir: cfg.Dir, shouldDownload: cfg.ShouldDownload, fetch: fetch}
	layout, err := files.resolve(ctx, cfg.Layout.Path, cfg.Layout.Name, "pp-doclayout.onnx")
	if err != nil {
		return nil, "", paths{}, newModelError("layout", cfg.Layout.Name, cfg.Layout.Path, err)
	}
	detect, err := files.resolve(ctx, cfg.Detect.Path, cfg.Detect.Name, "pp-ocr-det.onnx")
	if err != nil {
		return nil, "", paths{}, newModelError("detect", cfg.Detect.Name, cfg.Detect.Path, err)
	}
	recognize, err := files.resolve(ctx, cfg.Recognize.Path, cfg.Recognize.Name, "pp-ocr-rec.onnx")
	if err != nil {
		return nil, "", paths{}, newModelError("recognize", cfg.Recognize.Name, cfg.Recognize.Path, err)
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

func getModelName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func getCacheRoot() string {
	root, err := cache.GetSharedDir()
	if err != nil {
		return ""
	}
	return root
}

func newModelError(role, id, path string, err error) error {
	where := id
	if path != "" {
		where = path
	}
	return fmt.Errorf("the %s model %s is not available: %w; connect to the internet and run Recognize again", role, where, err)
}
