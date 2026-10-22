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
	engine, _, err := onnx.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, cfg.Dir))
	if err != nil {
		return nil, paths{}, err
	}
	found := paths{from: "settings"}
	if err := cfg.folder().Locate(ctx, pinnedSums, getWantedFiles(cfg, &found)); err != nil {
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
	underVal := func(name string) string {
		if cfg.Model.Repo == "" {
			return ""
		}
		return strings.TrimSuffix(cfg.Model.Repo, "/") + "/" + name
	}
	return []models.Wanted{
		{Dst: &into.encoder, Path: cfg.Model.Encoder, Name: underVal(encoderFile), Kind: "encoder"},
		{Dst: &into.decoder, Path: cfg.Model.Decoder, Name: underVal(decoderFile), Kind: "decoder"},
		{Dst: &into.joiner, Path: cfg.Model.Joiner, Name: underVal(joinerFile), Kind: "joiner"},
		{Dst: &into.tokens, Path: cfg.Model.Tokens, Name: underVal(tokensFile), Kind: "tokens"},
		{Dst: &into.segmenter, Path: cfg.Segmenter.Path, Name: cfg.Segmenter.Repo, Kind: "segmenter model"},
	}
}

// getModelName is what a model is called, for the record kept along with what it produced.
// A name in the settings stands, and a model without one is called after the
// file it was loaded from.
func getModelName(name, pos string) string {
	if name != "" {
		return name
	}
	return strings.TrimSuffix(path.Base(filepath.ToSlash(pos)), filepath.Ext(pos))
}

func (c Config) source() models.Source {
	return c.folder().Source()
}
