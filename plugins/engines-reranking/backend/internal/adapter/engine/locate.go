package engine

import (
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
)

const tokenizerFile = "tokenizer.json"

// FetchDependency resolves a model file that the manifest declares under an id through the host.
type FetchDependency func(ctx context.Context, id string) (string, error)

// Fetch is how the files of a model are obtained from the host: a shipped model by the ids its
// manifest declares, any other repository by address.
type Fetch struct {
	Dependency FetchDependency
	File       onnx.FetchFile
}

func Locate(ctx context.Context, p profile.Profile, fetch Fetch) (onnx.ModelFiles, error) {
	if len(p.Dependencies) > 0 {
		return fetchModelFiles(ctx, p, fetch.Dependency)
	}
	return onnx.PrepareModel(ctx, onnx.RepositoryModel{Endpoint: os.Getenv("HF_ENDPOINT"), ID: p.ID,
		Revision: p.Revision, Graph: p.OnnxFile, Sums: p.SHA256, Files: nil, SelectExternal: func(name string) bool { return strings.HasPrefix(name, p.OnnxFile) }}, getCacheRoot(), fetch.File)
}

func fetchModelFiles(ctx context.Context, p profile.Profile, fetch FetchDependency) (onnx.ModelFiles, error) {
	if _, ok := p.Dependencies[p.OnnxFile]; !ok {
		return onnx.ModelFiles{}, fmt.Errorf("%s declares no file %s", p.ID, p.OnnxFile)
	}
	locals := make(map[string]string, len(p.Dependencies))
	for _, file := range slices.Sorted(maps.Keys(p.Dependencies)) {
		local, err := fetch(ctx, p.Dependencies[file])
		if err != nil {
			return onnx.ModelFiles{}, fmt.Errorf("fetch %s of %s: %w", file, p.ID, err)
		}
		locals[file] = local
	}
	tokenizer, ok := locals[tokenizerFile]
	if !ok {
		return onnx.ModelFiles{}, fmt.Errorf("%s declares no %s", p.ID, tokenizerFile)
	}
	delete(locals, tokenizerFile)
	placed, err := onnx.PlaceModelFiles(getCacheRoot(), p.OnnxFile, locals)
	if err != nil {
		return onnx.ModelFiles{}, err
	}
	return onnx.ModelFiles{Model: placed, Tokenizer: tokenizer}, nil
}
