package engine

import (
	"context"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"os"
	"path"
	"path/filepath"
)

// Locate finds the model files: in a specified directory, or in the download cache.
func Locate(ctx context.Context, cfg model.LocalModel, fetch onnx.FetchFile) (onnx.ModelFiles, error) {
	file := cfg.File
	if file == "" {
		file = model.ModelFile
	}
	if cfg.Dir != "" {
		return locateInDir(cfg.Dir, file)
	}
	if cfg.Name == "" {
		return onnx.ModelFiles{}, errors.New("no model to run: name a repository or a directory")
	}
	if !cfg.ShouldDownload {
		return onnx.ModelFiles{}, fmt.Errorf("%s is not on this machine: set dir to where it is, or enable download", cfg.Name)
	}
	return onnx.PrepareModel(ctx, onnx.RepositoryModel{
		Endpoint: os.Getenv("HF_ENDPOINT"), ID: cfg.Name, Revision: cfg.Revision,
		Graph: path.Join(model.ModelFolder, file), Sums: cfg.Sums,
	}, getCacheRoot(), fetch)
}

func locateInDir(dir, file string) (onnx.ModelFiles, error) {
	p := onnx.ModelFiles{
		Model:     filepath.Join(dir, file),
		Tokenizer: filepath.Join(dir, model.TokenizerFile),
	}
	for _, required := range []string{p.Model, p.Tokenizer} {
		if _, err := os.Stat(required); err != nil {
			return onnx.ModelFiles{}, fmt.Errorf("model directory %s: %w", dir, err)
		}
	}
	return p, nil
}

func getCacheRoot() string {
	root, err := cache.GetSharedDirOrPluginDir()
	if err != nil {
		return ""
	}
	return root
}
