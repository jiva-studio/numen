package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/shared/rerank"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/usecase/reranking"
)

// reranker is the Reranker role of the SDK over the use case: it reads the request message and writes the
// result message. The instruction of the request is not passed on, because the model takes none.
type reranker struct {
	reranking *reranking.Reranker
}

func (r reranker) Rerank(ctx context.Context, call connection.Call, request *rerank.RerankRequest) (*rerank.RerankResult, error) {
	scores, err := r.reranking.Rerank(ctx, call, request.GetQuery(), request.GetDocuments())
	if err != nil {
		return nil, err
	}
	return &rerank.RerankResult{Scores: scores}, nil
}
