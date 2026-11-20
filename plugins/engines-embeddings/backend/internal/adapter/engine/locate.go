package engine

import (
	"context"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
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
	graph := path.Join(model.ModelFolder, file)
	files, err := getShippedFiles(cfg.Name, cfg.Revision, graph)
	if err != nil {
		return onnx.ModelFiles{}, err
	}
	return onnx.PrepareModel(ctx, onnx.RepositoryModel{
		Endpoint: os.Getenv("HF_ENDPOINT"), ID: cfg.Name, Revision: cfg.Revision,
		Graph: graph, Sums: cfg.Sums, Files: files,
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

func getShippedFiles(id, revision, graph string) ([]string, error) {
	table, err := model.GetShippedProfiles()
	if err != nil {
		return nil, err
	}
	for _, p := range table.GetCurated() {
		if p.ID == id && p.Revision == revision && p.OnnxFile == graph {
			return slices.Sorted(maps.Keys(p.SHA256)), nil
		}
	}
	return nil, nil
}
