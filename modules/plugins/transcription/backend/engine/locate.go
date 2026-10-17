package engine

import (
	"context"
	"path"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
	"github.com/jiva-studio/numen/modules/plugins/transcription/backend/onnxruntime"
)

// onnxRuntime is the ONNX Runtime this run listens onnxRuntime, and where it came
// from. It is the process's, opened once and held for as long as the process
// lives, so nothing here closes it.
type onnxRuntime struct {
	engine *ort.Engine
	at     string
}

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
func locateModels(ctx context.Context, cfg Config) (onnxRuntime, paths, error) {
	var opened onnxRuntime
	var err error
	if opened.engine, opened.at, err = onnxruntime.Open(ctx, cfg.settings()); err != nil {
		return onnxRuntime{}, paths{}, err
	}

	found := paths{from: "settings"}
	for _, one := range getWantedFiles(cfg, &found) {
		if *one.dst, err = models.Locate(ctx, cfg.source(), pinnedSums, one.path, one.name, one.kind); err != nil {
			return onnxRuntime{}, paths{}, err
		}
	}
	if cfg.Dir != "" {
		found.from = cfg.Dir
	}
	return opened, found, nil
}

// A wantedFile is one of the things a transcription reads: where it is put once
// it is found, where the settings say it is, where it is fetched from, and what
// it is called when it is missing.
type wantedFile struct {
	dst              *string
	path, name, kind string
}

// getWantedFiles is every file a transcription reads. The transducer is four of them,
// published as four names in one folder.
func getWantedFiles(cfg Config, into *paths) []wantedFile {
	under := func(name string) string {
		if cfg.Model.Repo == "" {
			return ""
		}
		return strings.TrimSuffix(cfg.Model.Repo, "/") + "/" + name
	}
	return []wantedFile{
		{&into.encoder, cfg.Model.Encoder, under(encoderFile), "encoder"},
		{&into.decoder, cfg.Model.Decoder, under(decoderFile), "decoder"},
		{&into.joiner, cfg.Model.Joiner, under(joinerFile), "joiner"},
		{&into.tokens, cfg.Model.Tokens, under(tokensFile), "tokens"},
		{&into.segmenter, cfg.Segmenter.Path, cfg.Segmenter.Repo, "segmenter model"},
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
