package engine

import (
	"context"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
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
	found := paths{from: "settings"}
	if err := cfg.folder().Locate(ctx, pinnedSums, []Wanted{
		{Dst: &found.layout, Path: cfg.Layout.Path, Name: cfg.Layout.Name, Kind: "layout model"},
		{Dst: &found.detect, Path: cfg.Detect.Path, Name: cfg.Detect.Name, Kind: "detect model"},
		{Dst: &found.recognise, Path: cfg.Recognise.Path, Name: cfg.Recognise.Name, Kind: "recognise model"},
	}); err != nil {
		return nil, "", paths{}, err
	}
	if cfg.Dir != "" {
		found.from = cfg.Dir
	}
	return engine, pos, found, nil
}

func name(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func (c Config) source() Source {
	return c.folder().Source()
}
