package recognition

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

// through is the ONNX Runtime this run reads its models through, and where it
// came from. It is the process's, opened once and held for as long as the
// process lives, so nothing here closes it.
type through struct {
	engine *ort.Engine
	at     string
}

// paths are the files this run reads its models out of, and where they were
// found. Nothing here is open: they are names until something reads them.
type paths struct {
	layout    string
	detect    string
	recognise string
	from      string
}

// locate opens the runtime and finds the models.
//
// Three places are tried in order and each is a setting: a path written down,
// the folder the application was installed into, and what was downloaded. A
// path that is written down is used as given, and its absence is an error
// rather than a reason to look elsewhere — a person who said where a model is
// meant it.
func locate(ctx context.Context, cfg Config) (through, paths, error) {
	var opened through
	var err error
	if opened.engine, opened.at, err = onnxruntime.Open(ctx, cfg.settings()); err != nil {
		return through{}, paths{}, err
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
			return through{}, paths{}, err
		}
	}
	if cfg.Dir != "" {
		found.from = cfg.Dir
	}
	return opened, found, nil
}

// model is where one model's file is.
//
// A path written down is used as given, and its absence is an error rather than
// a reason to look elsewhere: a person who said where a model is meant it. A
// name is looked for beside the application and then fetched.
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
	for _, at := range onnxruntime.Beside(cfg.Dir, filepath.Base(name)) {
		if _, err := os.Stat(at); err == nil {
			return at, nil
		}
	}
	if !onnxruntime.IsAddress(name) {
		return "", fmt.Errorf("the %s model %q is not beside the application, and is not somewhere to fetch it from", kind, name)
	}
	found, err := onnxruntime.Fetched(ctx, cfg.settings(), name)
	if err != nil {
		return "", fmt.Errorf("the %s model: %w", kind, err)
	}
	return found, nil
}

// name is what a model is called, for the record kept beside what it produced.
func name(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}
