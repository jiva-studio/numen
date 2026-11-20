package engine

import (
	"context"
	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
	"os"
)

func Locate(ctx context.Context, p profile.Profile, fetch onnx.FetchFile) (onnx.ModelFiles, error) {
	return onnx.PrepareModel(ctx, onnx.RepositoryModel{Endpoint: os.Getenv("HF_ENDPOINT"), ID: p.ID,
		Revision: p.Revision, Graph: p.OnnxFile, Sums: p.SHA256}, getCacheRoot(), fetch)
}
