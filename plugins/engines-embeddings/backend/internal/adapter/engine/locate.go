package engine

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

// FetchDependency resolves a model file that the manifest declares under an id through the host.
type FetchDependency func(ctx context.Context, id string) (string, error)

// Fetch is how the files of a model are obtained from the host: a shipped model by the ids its
// manifest declares, any other repository by address.
type Fetch struct {
	Dependency FetchDependency
	File       onnx.FetchFile
}

// Locate finds the model files: in a specified directory, or in the download cache.
func Locate(ctx context.Context, cfg model.LocalModel, fetch Fetch) (onnx.ModelFiles, error) {
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
	if len(cfg.Dependencies) > 0 {
		return fetchModelFiles(ctx, cfg.Name, graph, cfg.Dependencies, fetch.Dependency)
	}
	return onnx.PrepareModel(ctx, onnx.RepositoryModel{
		Endpoint: os.Getenv("HF_ENDPOINT"), ID: cfg.Name, Revision: cfg.Revision,
		Graph: graph, Sums: cfg.Sums, Files: nil,
	}, getCacheRoot(), fetch.File)
}

func fetchModelFiles(ctx context.Context, name, graph string, dependencies map[string]string, fetch FetchDependency) (onnx.ModelFiles, error) {
	if _, ok := dependencies[graph]; !ok {
		return onnx.ModelFiles{}, fmt.Errorf("%s declares no file %s", name, graph)
	}
	locals := make(map[string]string, len(dependencies))
	for _, file := range slices.Sorted(maps.Keys(dependencies)) {
		local, err := fetch(ctx, dependencies[file])
		if err != nil {
			return onnx.ModelFiles{}, fmt.Errorf("fetch %s of %s: %w", file, name, err)
		}
		locals[file] = local
	}
	tokenizer, ok := locals[model.TokenizerFile]
	if !ok {
		return onnx.ModelFiles{}, fmt.Errorf("%s declares no %s", name, model.TokenizerFile)
	}
	delete(locals, model.TokenizerFile)
	placed, err := onnx.PlaceModelFiles(getCacheRoot(), graph, locals)
	if err != nil {
		return onnx.ModelFiles{}, err
	}
	return onnx.ModelFiles{Model: placed, Tokenizer: tokenizer}, nil
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
	root, err := cache.GetSharedDir()
	if err != nil {
		return ""
	}
	return root
}
