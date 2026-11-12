package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/usecase/serving"
)

// handler answers the calls of the embedding service with the model server.
type handler struct {
	model *serving.Server
}

// GetModelInfo describes the model the plugin serves.
func (h handler) GetModelInfo(_ context.Context) (embedding.ModelInfo, error) {
	info, err := h.model.GetModelInfo()
	if err != nil {
		return embedding.ModelInfo{}, getRPCError(err)
	}
	return embedding.ModelInfo{
		Name:           info.Name,
		Dimensions:     int32(info.Dimensions),
		MaxTokens:      int32(info.MaxTokens),
		Pooling:        info.Pooling,
		From:           info.From,
		Concurrency:    int32(info.Concurrency),
		FloorHint:      info.FloorHint,
		MinChunkLength: int32(info.MinChunkLength),
		ChunkTokens:    int32(info.ChunkTokens),
		Handling:       info.Handling,
	}, nil
}

// EmbedTexts returns one vector per text.
func (h handler) EmbedTexts(ctx context.Context, role embedding.Role, texts []string) ([][]float32, error) {
	vectors, err := h.model.EmbedTexts(ctx, getEngineRole(role), texts)
	if err != nil {
		return nil, getRPCError(err)
	}
	return vectors, nil
}

func getEngineRole(role embedding.Role) model.Role {
	if role == embedding.RoleQuery {
		return model.RoleQuery
	}
	return model.RoleDocument
}

func getRPCError(err error) error {
	if failure, isFailure := errors.AsType[serving.PreconditionError](err); isFailure && failure.Err != nil {
		return rpcerror.New(rpcerror.FailedPrecondition, err)
	}
	return rpcerror.New(rpcerror.Internal, err)
}
