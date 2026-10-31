package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

type paths struct {
	layout    string
	detect    string
	recognise string
	from      string
}

func locateModels(ctx context.Context, cfg Config) (*ort.Engine, string, paths, error) {
	engine, pos, err := onnx.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, cfg.Dir))
	if err != nil {
		return nil, "", paths{}, err
	}
	layout, err := onnx.LocateModel(cfg.Layout.Path, "NUMEN_DEPENDENCY_PP_DOCLAYOUT", "pp-doclayout.onnx", cfg.Dir)
	if err != nil {
		return nil, "", paths{}, fmt.Errorf("layout model: %w", err)
	}
	detect, err := onnx.LocateModel(cfg.Detect.Path, "NUMEN_DEPENDENCY_PP_OCR_DET", "pp-ocr-det.onnx", cfg.Dir)
	if err != nil {
		return nil, "", paths{}, fmt.Errorf("detect model: %w", err)
	}
	recognise, err := onnx.LocateModel(cfg.Recognise.Path, "NUMEN_DEPENDENCY_PP_OCR_REC", "pp-ocr-rec.onnx", cfg.Dir)
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
