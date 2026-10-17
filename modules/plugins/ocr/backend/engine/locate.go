package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
	"github.com/jiva-studio/numen/modules/plugins/ocr/backend/onnxruntime"
)

type onnxRuntime struct {
	engine *ort.Engine
	at     string
}

type paths struct {
	layout    string
	detect    string
	recognise string
	from      string
}

func locateModels(ctx context.Context, cfg Config) (onnxRuntime, paths, error) {
	var opened onnxRuntime
	var err error
	if opened.engine, opened.at, err = onnxruntime.Open(cfg.Runtime); err != nil {
		return onnxRuntime{}, paths{}, err
	}

	found := paths{from: "settings"}
	for _, one := range []struct {
		dst              *string
		path, name, kind string
	}{
		{&found.layout, cfg.Layout.Path, cfg.Layout.Name, "layout"},
		{&found.detect, cfg.Detect.Path, cfg.Detect.Name, "detect"},
		{&found.recognise, cfg.Recognise.Path, cfg.Recognise.Name, "recognise"},
	} {
		if *one.dst, err = model(ctx, cfg, one.path, one.name, one.kind); err != nil {
			return onnxRuntime{}, paths{}, err
		}
	}
	if cfg.Dir != "" {
		found.from = cfg.Dir
	}
	return opened, found, nil
}

func model(ctx context.Context, cfg Config, path, name, kind string) (string, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("the %s model: %w", kind, err)
		}
		return path, nil
	}
	if name == "" {
		return "", fmt.Errorf("no %s model: name one, or say where it is", kind)
	}
	if at, ok := findLocal(cfg, name); ok {
		return at, nil
	}
	sum := getPinnedSum(name)
	if !models.IsAddress(name) {
		return "", fmt.Errorf("the %s model %q is not beside the application, and is not somewhere to fetch it from", kind, name)
	}
	found, err := cfg.source().Fetch(ctx, name, sum)
	if err != nil {
		return "", fmt.Errorf("the %s model: %w", kind, err)
	}
	return found, nil
}

func name(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// findLocal returns the file of a model that is beside the application or in
// the folder the settings name. A model with a pinned sum is found only when
// the file holds that sum.
func findLocal(cfg Config, name string) (string, bool) {
	sum := getPinnedSum(name)
	for _, at := range models.GetSearchPaths(cfg.Dir, filepath.Base(name)) {
		if _, err := os.Stat(at); err != nil {
			continue
		}
		if sum == "" || models.VerifyFile(at, sum) == nil {
			return at, true
		}
	}
	return "", false
}
