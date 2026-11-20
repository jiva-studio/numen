package engine

import (
	"context"
	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
	"maps"
	"os"
	"slices"
	"strings"
)

func Locate(ctx context.Context, p profile.Profile, fetch onnx.FetchFile) (onnx.ModelFiles, error) {
	files, err := getShippedFiles(p.ID, p.Revision, p.OnnxFile)
	if err != nil {
		return onnx.ModelFiles{}, err
	}
	return onnx.PrepareModel(ctx, onnx.RepositoryModel{Endpoint: os.Getenv("HF_ENDPOINT"), ID: p.ID,
		Revision: p.Revision, Graph: p.OnnxFile, Sums: p.SHA256, Files: files, SelectExternal: func(name string) bool { return strings.HasPrefix(name, p.OnnxFile) }}, getCacheRoot(), fetch)
}

func getShippedFiles(id, revision, graph string) ([]string, error) {
	table, err := profile.GetShippedProfiles()
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
