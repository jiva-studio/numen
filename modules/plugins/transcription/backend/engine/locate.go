package engine

import (
	"context"
	"path"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
)

// paths are the files this run listens through, and where they were found.
// Nothing here is open: they are names until something reads them.
type paths struct {
	encoder   string
	decoder   string
	joiner    string
	tokens    string
	segmenter string
	from      string
}

// locateModels opens the runtime and finds the models.
//
// Three places are tried in order and each is a setting: a path written down,
// the folder the application was installed into, and what was downloaded. A
// path that is written down is used as given, and its absence is an error.
func locateModels(ctx context.Context, cfg Config) (*ort.Engine, paths, error) {
	found := paths{from: "settings"}
	engine, _, err := onnx.OpenAndLocate(ctx, cfg.Runtime, cfg.folder(), pinnedSums, getWantedFiles(cfg, &found))
	if err != nil {
		return nil, paths{}, err
	}
	if cfg.Dir != "" {
		found.from = cfg.Dir
	}
	return engine, found, nil
}

// getWantedFiles is every file a transcription reads. The transducer is four of them,
// published as four names in one folder.
func getWantedFiles(cfg Config, into *paths) []models.Wanted {
	under := func(name string) string {
		if cfg.Model.Repo == "" {
			return ""
		}
		return strings.TrimSuffix(cfg.Model.Repo, "/") + "/" + name
	}
	return []models.Wanted{
		{Dst: &into.encoder, Path: cfg.Model.Encoder, Name: under(encoderFile), Kind: "encoder"},
		{Dst: &into.decoder, Path: cfg.Model.Decoder, Name: under(decoderFile), Kind: "decoder"},
		{Dst: &into.joiner, Path: cfg.Model.Joiner, Name: under(joinerFile), Kind: "joiner"},
		{Dst: &into.tokens, Path: cfg.Model.Tokens, Name: under(tokensFile), Kind: "tokens"},
		{Dst: &into.segmenter, Path: cfg.Segmenter.Path, Name: cfg.Segmenter.Repo, Kind: "segmenter model"},
	}
}

// getModelName is what a model is called, for the record kept beside what it produced.
// A name in the settings stands, and a model without one is called after the
// file it was loaded from.
func getModelName(name, at string) string {
	if name != "" {
		return name
	}
	return strings.TrimSuffix(path.Base(filepath.ToSlash(at)), filepath.Ext(at))
}

func (c Config) source() models.Source {
	return c.folder().Source()
}
