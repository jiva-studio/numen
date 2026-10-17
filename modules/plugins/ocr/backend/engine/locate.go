package engine

import (
	"context"
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
	if opened.engine, opened.at, err = onnxruntime.Open(ctx, cfg.settings()); err != nil {
		return onnxRuntime{}, paths{}, err
	}

	found := paths{from: "settings"}
	for _, one := range []struct {
		dst              *string
		path, name, kind string
	}{
		{&found.layout, cfg.Layout.Path, cfg.Layout.Name, "layout model"},
		{&found.detect, cfg.Detect.Path, cfg.Detect.Name, "detect model"},
		{&found.recognise, cfg.Recognise.Path, cfg.Recognise.Name, "recognise model"},
	} {
		if *one.dst, err = models.Locate(ctx, cfg.source(), pinnedSums, one.path, one.name, one.kind); err != nil {
			return onnxRuntime{}, paths{}, err
		}
	}
	if cfg.Dir != "" {
		found.from = cfg.Dir
	}
	return opened, found, nil
}

func name(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func (c Config) source() models.Source {
	return c.folder().Source()
}
